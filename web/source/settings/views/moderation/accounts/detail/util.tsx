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

import { useMemo } from "react";

import { AdminAccount } from "../../../../lib/types/account";
import { store } from "../../../../redux/store";

export function yesOrNo(b: boolean): string {
	return b ? "yes" : "no";
}

export function UseOurInstanceAccount(account: AdminAccount): boolean {
	// Pull our own URL out of storage so we can
	// tell if account is our instance account.
	const ourDomain = useMemo(() => {
		const instanceUrlStr = store.getState().oauth.instanceUrl;
		if (!instanceUrlStr) {
			return "";
		}

		const instanceUrl = new URL(instanceUrlStr);
		return instanceUrl.host;
	}, []);

	return !account.domain && account.username == ourDomain;
}
