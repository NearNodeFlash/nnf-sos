/*
 * Copyright 2023 Hewlett Packard Enterprise Development LP
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

package v1alpha2

func (*ClientMount) Hub()               {}
func (*Computes) Hub()                  {}
func (*DWDirectiveRule) Hub()           {}
func (*DirectiveBreakdown) Hub()        {}
func (*PersistentStorageInstance) Hub() {}
func (*Servers) Hub()                   {}
func (*Storage) Hub()                   {}
func (*SystemConfiguration) Hub()       {}
func (*Workflow) Hub()                  {}

// The conversion-verifier tool wants these...though they're never used.
func (*ClientMountList) Hub()               {}
func (*ComputesList) Hub()                  {}
func (*DWDirectiveRuleList) Hub()           {}
func (*DirectiveBreakdownList) Hub()        {}
func (*PersistentStorageInstanceList) Hub() {}
func (*ServersList) Hub()                   {}
func (*StorageList) Hub()                   {}
func (*SystemConfigurationList) Hub()       {}
func (*WorkflowList) Hub()                  {}
