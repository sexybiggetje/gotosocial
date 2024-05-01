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

import { MenuItem } from "../../lib/navigation/menu";
import React from "react";

/**
 * - /settings/user/profile
 * - /settings/user/settings
 * - /settings/user/migration
 */
export default function UserMenu() {	
	return (
		<MenuItem
			name="User"
			itemUrl="user"
			defaultChild="profile"
		>
			<MenuItem
				name="Profile"
				itemUrl="profile"
				icon="fa-user"
			/>
			<MenuItem
				name="Settings"
				itemUrl="settings"
				icon="fa-cogs"
			/>
			<MenuItem
				name="Migration"
				itemUrl="migration"
				icon="fa-exchange"
			/>
		</MenuItem>
	);
}