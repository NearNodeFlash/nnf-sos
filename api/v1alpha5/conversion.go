/*
 * Copyright 2024-2025 Hewlett Packard Enterprise Development LP
 * Other additional copyright holders may be indicated within.
 *
 * The entirety of this work is licensed under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 *
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha5

func (*NnfAccess) Hub()              {}
func (*NnfContainerProfile) Hub()    {}
func (*NnfDataMovement) Hub()        {}
func (*NnfDataMovementManager) Hub() {}
func (*NnfDataMovementProfile) Hub() {}
func (*NnfLustreMGT) Hub()           {}
func (*NnfNode) Hub()                {}
func (*NnfNodeBlockStorage) Hub()    {}
func (*NnfNodeECData) Hub()          {}
func (*NnfNodeStorage) Hub()         {}
func (*NnfPortManager) Hub()         {}
func (*NnfStorage) Hub()             {}
func (*NnfStorageProfile) Hub()      {}
func (*NnfSystemStorage) Hub()       {}

// The conversion-verifier tool wants these...though they're never used.
func (*NnfAccessList) Hub()              {}
func (*NnfContainerProfileList) Hub()    {}
func (*NnfDataMovementList) Hub()        {}
func (*NnfDataMovementManagerList) Hub() {}
func (*NnfDataMovementProfileList) Hub() {}
func (*NnfLustreMGTList) Hub()           {}
func (*NnfNodeList) Hub()                {}
func (*NnfNodeBlockStorageList) Hub()    {}
func (*NnfNodeECDataList) Hub()          {}
func (*NnfNodeStorageList) Hub()         {}
func (*NnfPortManagerList) Hub()         {}
func (*NnfStorageList) Hub()             {}
func (*NnfStorageProfileList) Hub()      {}
func (*NnfSystemStorageList) Hub()       {}
