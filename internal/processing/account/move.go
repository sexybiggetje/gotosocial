// GoToSocial
// Copyright (C) GoToSocial Authors admin@gotosocial.org
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package account

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/superseriousbusiness/gotosocial/internal/ap"
	apimodel "github.com/superseriousbusiness/gotosocial/internal/api/model"
	"github.com/superseriousbusiness/gotosocial/internal/federation/dereferencing"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/id"
	"github.com/superseriousbusiness/gotosocial/internal/messages"
	"github.com/superseriousbusiness/gotosocial/internal/oauth"
	"github.com/superseriousbusiness/gotosocial/internal/uris"
	"golang.org/x/crypto/bcrypt"
)

func (p *Processor) MoveSelf(
	ctx context.Context,
	authed *oauth.Auth,
	form *apimodel.AccountMoveRequest,
) gtserror.WithCode {
	// Ensure valid MovedToURI.
	if form.MovedToURI == "" {
		err := errors.New("no moved_to_uri provided in account Move request")
		return gtserror.NewErrorBadRequest(err, err.Error())
	}

	targetAcctURIStr := form.MovedToURI
	targetAcctURI, err := url.Parse(form.MovedToURI)
	if err != nil {
		err := fmt.Errorf("invalid moved_to_uri provided in account Move request: %w", err)
		return gtserror.NewErrorBadRequest(err, err.Error())
	}

	if targetAcctURI.Scheme != "https" && targetAcctURI.Scheme != "http" {
		err := errors.New("invalid moved_to_uri provided in account Move request: uri scheme must be http or https")
		return gtserror.NewErrorBadRequest(err, err.Error())
	}

	// Self account Move requires password to ensure it's for real.
	if form.Password == "" {
		err := errors.New("no password provided in account Move request")
		return gtserror.NewErrorBadRequest(err, err.Error())
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(authed.User.EncryptedPassword),
		[]byte(form.Password),
	); err != nil {
		err := errors.New("invalid password provided in account Move request")
		return gtserror.NewErrorBadRequest(err, err.Error())
	}

	// We can't/won't validate Move activities
	// to domains we have blocked, so check this.
	targetDomainBlocked, err := p.state.DB.IsDomainBlocked(ctx, targetAcctURI.Host)
	if err != nil {
		err := fmt.Errorf(
			"db error checking if target domain %s blocked: %w",
			targetAcctURI.Host, err,
		)
		return gtserror.NewErrorInternalError(err)
	}

	if targetDomainBlocked {
		err := fmt.Errorf(
			"domain of %s is blocked from this instance; "+
				"you will not be able to Move to that account",
			targetAcctURIStr,
		)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	var (
		// Current account from which
		// the move is taking place.
		originAcct = authed.Account

		// Target account to which
		// the move is taking place.
		targetAcct *gtsmodel.Account

		// AP representation of target.
		targetAcctable ap.Accountable
	)

	// Next steps involve checking + setting
	// state that might get messed up if a
	// client triggers this function twice
	// in quick succession, so get a lock on
	// this account.
	lockKey := originAcct.URI
	unlock := p.state.ProcessingLocks.Lock(lockKey)
	defer unlock()

	// Ensure we have a valid, up-to-date representation of the target account.
	targetAcct, targetAcctable, err = p.federator.GetAccountByURI(
		ctx,
		originAcct.Username,
		targetAcctURI,
	)
	if err != nil {
		err := fmt.Errorf("error dereferencing moved_to_uri account: %w", err)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	if !targetAcct.SuspendedAt.IsZero() {
		err := fmt.Errorf(
			"target account %s is suspended from this instance; "+
				"you will not be able to Move to that account",
			targetAcct.URI,
		)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	if targetAcct.IsRemote() {
		// Force refresh Move target account
		// to ensure we have up-to-date version.
		targetAcct, _, err = p.federator.RefreshAccount(ctx,
			originAcct.Username,
			targetAcct,
			targetAcctable,
			dereferencing.Freshest,
		)
		if err != nil {
			err := fmt.Errorf(
				"error refreshing target account %s: %w",
				targetAcctURIStr, err,
			)
			return gtserror.NewErrorUnprocessableEntity(err, err.Error())
		}
	}

	// If originAcct has already moved, ensure
	// this move reattempt is to the same account.
	if originAcct.IsMoving() &&
		originAcct.MovedToURI != targetAcct.URI {
		err := fmt.Errorf(
			"your account is already Moving or has Moved to %s; you cannot also Move to %s",
			originAcct.MovedToURI, targetAcct.URI,
		)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	// Target account MUST be aliased to this
	// account for this to be a valid Move.
	if !slices.Contains(targetAcct.AlsoKnownAsURIs, originAcct.URI) {
		err := fmt.Errorf(
			"target account %s is not aliased to this account via alsoKnownAs; "+
				"if you just changed it, please wait a few minutes and try the Move again",
			targetAcct.URI,
		)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	// Target account cannot itself have
	// already Moved somewhere else.
	if targetAcct.MovedToURI != "" {
		err := fmt.Errorf(
			"target account %s has already Moved somewhere else (%s); "+
				"you will not be able to Move to that account",
			targetAcct.URI, targetAcct.MovedToURI,
		)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	// If a Move has been *attempted* within last 5m,
	// that involved the origin and target in any way,
	// then we shouldn't try to reprocess immediately.
	latestMoveAttempt, err := p.state.DB.GetLatestMoveAttemptInvolvingURIs(
		ctx, originAcct.URI, targetAcct.URI,
	)
	if err != nil {
		err := fmt.Errorf(
			"error checking latest Move attempt involving origin %s and target %s: %w",
			originAcct.URI, targetAcct.URI, err,
		)
		return gtserror.NewErrorInternalError(err)
	}

	if !latestMoveAttempt.IsZero() &&
		time.Since(latestMoveAttempt) < 5*time.Minute {
		err := fmt.Errorf(
			"your account or target account have been involved in a Move attempt within "+
				"the last 5 minutes, will not process Move; please try again after %s",
			latestMoveAttempt.Add(5*time.Minute),
		)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	// If a Move has *succeeded* within the last week
	// that involved the origin and target in any way,
	// then we shouldn't process again for a while.
	latestMoveSuccess, err := p.state.DB.GetLatestMoveSuccessInvolvingURIs(
		ctx, originAcct.URI, targetAcct.URI,
	)
	if err != nil {
		err := fmt.Errorf(
			"error checking latest Move success involving origin %s and target %s: %w",
			originAcct.URI, targetAcct.URI, err,
		)
		return gtserror.NewErrorInternalError(err)
	}

	if !latestMoveSuccess.IsZero() &&
		time.Since(latestMoveSuccess) < 168*time.Hour {
		err := fmt.Errorf(
			"your account or target account have been involved in a successful Move within "+
				"the last 7 days, will not process Move; please try again after %s",
			latestMoveSuccess.Add(168*time.Hour),
		)
		return gtserror.NewErrorUnprocessableEntity(err, err.Error())
	}

	// See if we have a Move stored already
	// or if we need to create a new one.
	var move *gtsmodel.Move

	if originAcct.MoveID != "" {
		// Move already stored, ensure it's
		// to the target and nothing weird is
		// happening with race conditions etc.
		move = originAcct.Move
		if move == nil {
			// This shouldn't happen...
			err := fmt.Errorf("nil move for id %s", originAcct.MoveID)
			return gtserror.NewErrorInternalError(err)
		}

		if move.OriginURI != originAcct.URI ||
			move.TargetURI != targetAcct.URI {
			// This is also weird...
			err := errors.New("a Move is already stored for your account but contains invalid fields")
			return gtserror.NewErrorUnprocessableEntity(err, err.Error())
		}

		if originAcct.MovedToURI != move.TargetURI {
			// Huh... I'll be damned.
			err := errors.New("stored Move target URI does not equal your moved_to_uri value")
			return gtserror.NewErrorUnprocessableEntity(err, err.Error())
		}
	} else {
		// Move not stored yet, create it.
		moveID := id.NewULID()
		moveURIStr := uris.GenerateURIForMove(originAcct.Username, moveID)

		// We might have selected the target
		// using the URL and not the URI.
		// Ensure we continue with the URI!
		if targetAcctURIStr != targetAcct.URI {
			targetAcctURIStr = targetAcct.URI
			targetAcctURI, err = url.Parse(targetAcctURIStr)
			if err != nil {
				return gtserror.NewErrorInternalError(err)
			}
		}

		// Parse origin URI.
		originAcctURI, err := url.Parse(originAcct.URI)
		if err != nil {
			return gtserror.NewErrorInternalError(err)
		}

		// Store the Move.
		move = &gtsmodel.Move{
			ID:          moveID,
			AttemptedAt: time.Now(),
			OriginURI:   originAcct.URI,
			Origin:      originAcctURI,
			TargetURI:   targetAcctURIStr,
			Target:      targetAcctURI,
			URI:         moveURIStr,
		}
		if err := p.state.DB.PutMove(ctx, move); err != nil {
			err := fmt.Errorf("db error storing move %s: %w", moveURIStr, err)
			return gtserror.NewErrorInternalError(err)
		}

		// Update account with the new
		// Move, and set moved_to_uri.
		originAcct.MoveID = move.ID
		originAcct.Move = move
		originAcct.MovedToURI = targetAcct.URI
		originAcct.MovedTo = targetAcct
		if err := p.state.DB.UpdateAccount(
			ctx,
			originAcct,
			"move_id",
			"moved_to_uri",
		); err != nil {
			err := fmt.Errorf("db error updating account: %w", err)
			return gtserror.NewErrorInternalError(err)
		}
	}

	// Everything seems OK, process Move side effects async.
	p.state.Workers.Client.Queue.Push(&messages.FromClientAPI{
		APObjectType:   ap.ActorPerson,
		APActivityType: ap.ActivityMove,
		GTSModel:       move,
		Origin:         originAcct,
		Target:         targetAcct,
	})

	return nil
}
