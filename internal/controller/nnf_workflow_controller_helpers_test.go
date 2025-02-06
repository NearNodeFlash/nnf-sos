package controller

import (
	"reflect"

	lusv1beta1 "github.com/NearNodeFlash/lustre-fs-operator/api/v1beta1"
	nnfv1alpha5 "github.com/NearNodeFlash/nnf-sos/api/v1alpha5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("NnfWorkflowControllerHelpers", func() {

	Context("splitStagingArgumentIntoNameAndPath", func() {
		DescribeTable("",
			func(arg, name, path string) {
				n, p := splitStagingArgumentIntoNameAndPath(arg)
				Expect(n).To(Equal(name))
				Expect(p).To(Equal(path))

			},
			Entry("DW_JOB - No name", "$DW_JOB", "", ""),
			Entry("DW_JOB - No name_", "$DW_JOB_", "", ""),
			Entry("DW_JOB - Root", "$DW_JOB_hello", "hello", ""),
			Entry("DW_JOB - Root with trailing slash", "$DW_JOB_hi_there/", "hi-there", "/"),
			Entry("DW_JOB - Root/file", "$DW_JOB_my_gfs2/file", "my-gfs2", "/file"),
			Entry("DW_JOB - Root/dir/", "$DW_JOB_my_gfs2/dir/", "my-gfs2", "/dir/"),
			Entry("DW_JOB - Path/to/a/file", "$DW_JOB_my_file_system_name/path/to/a/file", "my-file-system-name", "/path/to/a/file"),
			Entry("DW_JOB - Path/to/a/dir/", "$DW_JOB_my_file_system_name/path/to/a/dir/", "my-file-system-name", "/path/to/a/dir/"),

			Entry("DW_PERSISTENT - No name", "$DW_PERSISTENT", "", ""),
			Entry("DW_PERSISTENT - No name_", "$DW_PERSISTENT_", "", ""),
			Entry("DW_PERSISTENT - Root", "$DW_PERSISTENT_hello", "hello", ""),
			Entry("DW_PERSISTENT - Root with trailing slash", "$DW_PERSISTENT_hi_there/", "hi-there", "/"),
			Entry("DW_PERSISTENT - Root/file", "$DW_PERSISTENT_my_gfs2/file", "my-gfs2", "/file"),
			Entry("DW_PERSISTENT - Root/dir/", "$DW_PERSISTENT_my_gfs2/dir/", "my-gfs2", "/dir/"),
			Entry("DW_PERSISTENT - Path/to/a/file", "$DW_PERSISTENT_my_file_system_name/path/to/a/file", "my-file-system-name", "/path/to/a/file"),
			Entry("DW_PERSISTENT - Path/to/a/dir/", "$DW_PERSISTENT_my_file_system_name/path/to/a/dir/", "my-file-system-name", "/path/to/a/dir/"),

			Entry("Global Lustre - empty", "", "", ""),
			Entry("Global Lustre - file", "/lus", "", "/lus"),
			Entry("Global Lustre - dir", "/lus/", "", "/lus/"),
			Entry("Global Lustre - user dir", "/lus/global/user", "", "/lus/global/user"),
		)
	})

	Context("getRabbitRelativePath", func() {

		DescribeTable("Test NNF filesystems (NnfAccess)",
			func(fsType, path, output string) {
				// We can hardwire these fields and assume the same mountpath/mountpathprefix, index, namespace, etc
				objRef := corev1.ObjectReference{Kind: reflect.TypeOf(nnfv1alpha5.NnfStorage{}).Name()}
				mntPath := "/mnt/nnf/123456-0/"
				idx := 0
				ns := "slushy44"

				access := nnfv1alpha5.NnfAccess{
					Spec: nnfv1alpha5.NnfAccessSpec{
						MountPath:       mntPath,
						MountPathPrefix: mntPath,
					},
				}

				Expect(getRabbitRelativePath(fsType, &objRef, &access, path, ns, idx)).To(Equal(output))
			},
			Entry("GFS2 - Root", "gfs2", "", "/mnt/nnf/123456-0/slushy44-0"),
			Entry("GFS2 - Root/", "gfs2", "/", "/mnt/nnf/123456-0/slushy44-0/"),
			Entry("GFS2 - Root/file", "gfs2", "/file", "/mnt/nnf/123456-0/slushy44-0/file"),
			Entry("GFS2 - Root/dir", "gfs2", "/dir", "/mnt/nnf/123456-0/slushy44-0/dir"),
			Entry("GFS2 - Root/dir/", "gfs2", "/dir/", "/mnt/nnf/123456-0/slushy44-0/dir/"),
			Entry("GFS2 - Path/to/a/file", "gfs2", "/path/to/a/file", "/mnt/nnf/123456-0/slushy44-0/path/to/a/file"),
			Entry("GFS2 - Path/to/a/dir/", "gfs2", "/path/to/a/dir/", "/mnt/nnf/123456-0/slushy44-0/path/to/a/dir/"),

			// these should all be the same as xfs, but why not?
			Entry("XFS - Root", "xfs", "", "/mnt/nnf/123456-0/slushy44-0"),
			Entry("XFS - Root/", "xfs", "/", "/mnt/nnf/123456-0/slushy44-0/"),
			Entry("XFS - Root/file", "xfs", "/file", "/mnt/nnf/123456-0/slushy44-0/file"),
			Entry("XFS - Root/dir", "xfs", "/dir", "/mnt/nnf/123456-0/slushy44-0/dir"),
			Entry("XFS - Root/dir/", "xfs", "/dir/", "/mnt/nnf/123456-0/slushy44-0/dir/"),
			Entry("XFS - Path/to/a/file", "xfs", "/path/to/a/file", "/mnt/nnf/123456-0/slushy44-0/path/to/a/file"),
			Entry("XFS - Path/to/a/dir/", "xfs", "/path/to/a/dir/", "/mnt/nnf/123456-0/slushy44-0/path/to/a/dir/"),

			// lustre doesn't have index mount directories
			Entry("Lustre - Root", "lustre", "", "/mnt/nnf/123456-0"),
			Entry("Lustre - Root/", "lustre", "/", "/mnt/nnf/123456-0/"),
			Entry("Lustre - Root/file", "lustre", "/file", "/mnt/nnf/123456-0/file"),
			Entry("Lustre - Root/dir", "lustre", "/dir", "/mnt/nnf/123456-0/dir"),
			Entry("Lustre - Root/dir/", "lustre", "/dir/", "/mnt/nnf/123456-0/dir/"),
			Entry("Lustre - Path/to/a/file", "lustre", "/path/to/a/file", "/mnt/nnf/123456-0/path/to/a/file"),
			Entry("Lustre - Path/to/a/dir/", "lustre", "/path/to/a/dir/", "/mnt/nnf/123456-0/path/to/a/dir/"),
		)

		DescribeTable("Test global lustre (Lustrefilesystem)",
			func(path, output string) {
				objRef := corev1.ObjectReference{
					Kind:      reflect.TypeOf(lusv1beta1.LustreFileSystem{}).Name(),
					Name:      "lustre2",
					Namespace: "default",
				}
				// idx, ns, and type only matters for NnfAccess object references, not for lustrefilesystems
				idx := 0
				ns := "slushy44"
				fsType := "none"

				Expect(getRabbitRelativePath(fsType, &objRef, nil, path, ns, idx)).To(Equal(output))
			},
			Entry("Global Lustre - user dir", "/lus/global/user", "/lus/global/user"),
			Entry("Global Lustre - empty", "", ""),
			Entry("Global Lustre - file", "/lus", "/lus"),
			Entry("Global Lustre - dir", "/lus/", "/lus/"),
		)
	})
})
