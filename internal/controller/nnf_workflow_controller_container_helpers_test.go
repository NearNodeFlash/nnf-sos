/*
 * Copyright 2024-2026 Hewlett Packard Enterprise Development LP
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

package controller

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("NnfWorkflowControllerContainerHelpers", func() {

	Context("addInitContainerPasswd", func() {

		var (
			container nnfUserContainer
			spec      corev1.PodSpec
		)

		BeforeEach(func() {
			container = nnfUserContainer{
				username: "mpiuser",
				uid:      1000,
				gid:      1000,
			}
			spec = corev1.PodSpec{}
		})

		It("should rewrite the home directory in the host entry branch", func() {
			container.addInitContainerPasswd(&spec, "test-image:latest")

			Expect(spec.InitContainers).To(HaveLen(1))
			script := spec.InitContainers[0].Command[2]

			// The host entry branch must rewrite field 6 (home dir) to /home/mpiuser
			// so it matches the SSH key mount path set by SSHAuthMountPath.
			Expect(script).To(ContainSubstring(`awk -F: -v home="/home/mpiuser"`))
			Expect(script).To(ContainSubstring(`{$6=home; print}`))

			// It must NOT echo HOST_ENTRY verbatim (that was the bug).
			Expect(script).NotTo(ContainSubstring("echo \"$HOST_ENTRY\" >> /config/passwd"))
		})

		It("should use the correct home directory in the fallback branch", func() {
			container.addInitContainerPasswd(&spec, "test-image:latest")

			script := spec.InitContainers[0].Command[2]

			// The fallback branch synthesizes a passwd entry with /home/mpiuser
			Expect(script).To(ContainSubstring("mpiuser:x:1000:1000::/home/mpiuser:/bin/sh"))
		})

		It("should substitute UID and GID values", func() {
			container.uid = 5555
			container.gid = 6666
			container.addInitContainerPasswd(&spec, "test-image:latest")

			script := spec.InitContainers[0].Command[2]

			// Verify Go's strings.ReplaceAll substituted the UID/GID
			Expect(script).To(ContainSubstring("5555"))
			Expect(script).To(ContainSubstring("6666"))
			Expect(script).NotTo(ContainSubstring("$UID"))
			Expect(script).NotTo(ContainSubstring("$GID"))
		})

		It("should not contain unreplaced $USER placeholders", func() {
			container.addInitContainerPasswd(&spec, "test-image:latest")

			script := spec.InitContainers[0].Command[2]

			// All $USER references should be replaced with the actual username.
			// Split on the username and check the remaining fragments for $USER.
			fragments := strings.Split(script, "mpiuser")
			for _, frag := range fragments {
				Expect(frag).NotTo(ContainSubstring("$USER"))
			}
		})

		It("should ensure both branches write to /config/passwd", func() {
			container.addInitContainerPasswd(&spec, "test-image:latest")

			script := spec.InitContainers[0].Command[2]

			// Both branches must write to /config/passwd (the EmptyDir volume)
			Expect(strings.Count(script, "/config/passwd")).To(BeNumerically(">=", 4))
		})
	})
})
