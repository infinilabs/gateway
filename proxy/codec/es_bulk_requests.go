// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Framework is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package codec

//data format:
//first line: action metadata
//optional second line: action body, delete action doesn't have body

//for example:
//{ "create" : { "_index" : "test-15","_type":"doc", "_id" : "carcr82i4h906lro3ai0" , "routing" : "carcr82i4h906lro3ahg" } }
//{ "id" : "carcr82i4h906lro3aig","routing_no" : "carcr82i4h906lro3ahg","batch_number" : "73632", "random_no" : "13","ip" : "not_found","now_local" : "2022-06-25 16:56:00.7871 +0800 CST","now_unix" : "1656147360" }
//{ "create" : { "_index" : "test-15","_type":"doc", "_id" : "carcr82i4h906lro3ajg" , "routing" : "carcr82i4h906lro3aj0" } }
//{ "id" : "carcr82i4h906lro3ak0","routing_no" : "carcr82i4h906lro3aj0","batch_number" : "73632", "random_no" : "15","ip" : "not_found","now_local" : "2022-06-25 16:56:00.787175 +0800 CST","now_unix" : "1656147360" }
