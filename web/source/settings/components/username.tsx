/*
	GoToSocial
	Copyright (C) GoToSocial Authors admin@gotosocial.org
	SPDX-License-Identifier: AGPL-3.0-or-later

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

import React from "react";
import { useLocation } from "wouter";
import { AdminAccount } from "../lib/types/account";

interface UsernameProps {
	account: AdminAccount;
	linkTo?: string;
	backLocation?: string;
	classNames?: string[];
}

export default function Username({ account, linkTo, backLocation, classNames }: UsernameProps) {
	const [ _location, setLocation ] = useLocation();
	
	let className = "username-lozenge";
	let isLocal = account.domain == null;

	if (account.suspended) {
		className += " suspended";
	}

	if (isLocal) {
		className += " local";
	}

	if (classNames) {
		className = [ className, classNames ].flat().join(" ");
	}

	let icon = isLocal
		? { fa: "fa-home", info: "Local user" }
		: { fa: "fa-external-link-square", info: "Remote user" };

	const content = (
		<>
			<i className={`fa fa-fw ${icon.fa}`} aria-hidden="true" title={icon.info} />
			<span className="sr-only">{icon.info}</span>
			&nbsp;
			<span className="acct">@{account.account.acct}</span>
		</>
	);

	if (linkTo) {
		className += " spanlink";
		return (
			<span
				className={className}
				onClick={() => {
					// When clicking on an account, direct
					// to the detail view for that account.
					setLocation(linkTo, {
						// Store the back location in history so
						// the detail view can use it to return to
						// this page (including query parameters).
						state: { backLocation: backLocation }
					});
				}}
				role="link"
				tabIndex={0}
			>
				{content}
			</span>
		);
	} else {
		return (
			<div className={className}>
				{content}
			</div>
		);
	}
}
