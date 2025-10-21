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

package v1alpha7

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NnfDataMovementProfileData defines the desired state of NnfDataMovementProfile
type NnfDataMovementProfileData struct {

	// Default is true if this instance is the default resource to use
	// +kubebuilder:default:=false
	Default bool `json:"default,omitempty"`

	// Pinned is true if this instance is an immutable copy
	// +kubebuilder:default:=false
	Pinned bool `json:"pinned,omitempty"`

	// Slots is the number of slots specified in the MPI hostfile. A value of 0 disables the use of
	// slots in the hostfile. The hostfile is used for both `statCommand` and `Command`.
	// +kubebuilder:default:=8
	// +kubebuilder:validation:Minimum:=0
	Slots int `json:"slots"`

	// MaxSlots is the number of max_slots specified in the MPI hostfile. A value of 0 disables the
	// use of max_slots in the hostfile. The hostfile is used for both `statCommand` and `Command`.
	// +kubebuilder:default:=0
	// +kubebuilder:validation:Minimum:=0
	MaxSlots int `json:"maxSlots"`

	// Command to execute to perform data movement. $VARS are replaced by the nnf software and must
	// be present in the command.
	// Available $VARS:
	//   HOSTFILE: hostfile that is created and used for mpirun. Contains a list of hosts and the
	//             slots/max_slots for each host. This hostfile is created at `/tmp/<dm-name>/hostfile`
	//   UID: User ID that is inherited from the Workflow
	//   GID: Group ID that is inherited from the Workflow
	//   SRC: source for the data movement
	//   DEST destination for the data movement
	// +kubebuilder:default:="ulimit -n 2048 && mpirun --allow-run-as-root --hostfile $HOSTFILE dcp --progress 1 --uid $UID --gid $GID $SRC $DEST"
	// +kubebuilder:validation:XValidation:rule=`!self.contains("dcp") || (self.contains("$HOSTFILE") && self.contains("$UID") && self.contains("$GID") && self.contains("$SRC") && self.contains("$DEST"))`
	Command string `json:"command"`

	// If true, enable the command's stdout to be saved in the log when the command completes
	// successfully. On failure, the output is always logged.
	// +kubebuilder:default:=false
	LogStdout bool `json:"logStdout,omitempty"`

	// Similar to logStdout, store the command's stdout in Status.Message when the command completes
	// successfully. On failure, the output is always stored.
	// +kubebuilder:default:=false
	StoreStdout bool `json:"storeStdout,omitempty"`

	// NnfDataMovement resources have the ability to collect and store the progress percentage and the
	// last few lines of output in the CommandStatus field. This number is used for the interval to collect
	// the progress data. `dcp --progress N` must be included in the data movement command in order for
	// progress to be collected. A value of 0 disables this functionality.
	// +kubebuilder:default:=5
	// +kubebuilder:validation:Minimum:=0
	ProgressIntervalSeconds int `json:"progressIntervalSeconds,omitempty"`

	// CreateDestDir will ensure that the destination directory exists before performing data
	// movement. This will cause a number of stat commands to determine the source and destination
	// file types, so that the correct pathing for the destination can be determined. Then, a mkdir
	// is issued.
	// +kubebuilder:default:=true
	CreateDestDir bool `json:"createDestDir"`

	// If CreateDestDir is true, then use StatCommand to perform the stat commands.
	// Use setpriv to execute with the specified UID/GID.
	// Available $VARS:
	//   HOSTFILE: Hostfile that is created and used for mpirun. Contains a list of hosts and the
	//             slots/max_slots for each host. This hostfile is created at
	//             `/tmp/<dm-name>/hostfile`. This is the same hostfile used as the one for Command.
	//   SETPRIV: Placeholder for where to inject the SETPRIV command to become the UID/GID
	//   		  inherited from the workflow.
	//   PATH: Path to stat
	// +kubebuilder:default:="mpirun --allow-run-as-root -np 1 --hostfile $HOSTFILE -- $SETPRIV stat --cached never -c '%F' $PATH"
	// +kubebuilder:validation:XValidation:rule=`!self.contains("stat") || (self.contains("$HOSTFILE") && self.contains("$SETPRIV") && self.contains("$PATH"))`
	StatCommand string `json:"statCommand"`

	// If CreateDestDir is true, then use MkdirCommand to perform the mkdir commands.
	// Use setpriv to execute with the specified UID/GID.
	// Available $VARS:
	//   HOSTFILE: Hostfile that is created and used for mpirun. Contains a list of hosts and the
	//             slots/max_slots for each host. This hostfile is created at
	//             `/tmp/<dm-name>/hostfile`. This is the same hostfile used as the one for Command.
	//   SETPRIV: Placeholder for where to inject the SETPRIV command to become the UID/GID
	//   		  inherited from the workflow.
	//   PATH: Path to stat
	// +kubebuilder:default:="mpirun --allow-run-as-root -np 1 --hostfile $HOSTFILE -- $SETPRIV mkdir -p $PATH"
	// +kubebuilder:validation:XValidation:rule=`!self.contains("mkdir") || (self.contains("$HOSTFILE") && self.contains("$SETPRIV") && self.contains("$PATH"))`
	MkdirCommand string `json:"mkdirCommand"`

	// The full setpriv command that is used to become the user and group specified in the workflow.
	// This is used by the StatCommand and MkdirCommand.
	// Available $VARS:
	//   UID: User ID that is inherited from the Workflow
	//   GID: Group ID that is inherited from the Workflow
	// +kubebuilder:default:="setpriv --euid $UID --egid $GID --clear-groups"
	// +kubebuilder:validation:XValidation:rule=`!self.contains("setpriv") || (self.contains("$UID") && self.contains("$GID"))`
	SetprivCommand string `json:"setprivCommand"`
}

// +kubebuilder:object:root=true
// +kubebuilder:unservedversion
// +kubebuilder:printcolumn:name="DEFAULT",type="boolean",JSONPath=".data.default",description="True if this is the default instance"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// NnfDataMovementProfile is the Schema for the nnfdatamovementprofiles API
type NnfDataMovementProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Data NnfDataMovementProfileData `json:"data,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:unservedversion

// NnfDataMovementProfileList contains a list of NnfDataMovementProfile
type NnfDataMovementProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NnfDataMovementProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NnfDataMovementProfile{}, &NnfDataMovementProfileList{})
}
