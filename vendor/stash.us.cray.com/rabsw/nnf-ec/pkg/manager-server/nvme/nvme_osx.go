// +build darwin

package nvme

import (
	"fmt"
)

func GetNamespaceId(path string) (int, error) {
	return -1, fmt.Errorf("Namespace ID Unsupported for OSX")
}
