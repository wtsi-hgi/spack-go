package spack

import (
	"fmt"
	"testing"
)

func TestSpackLoc(t *testing.T) {
	s := New("/software/singularity/3.11.4//bin/singularity", "run", "--bind", "/software", "/software/hgi/installs/softpack/spack/spack.sif", "--config", "config:install_tree:root:/software/hgi/installs/softpack/rstudio/.spack", "--config", "config:install_tree:padded_length:false")

	fmt.Println("START")
	fmt.Println(s.GetInstallRoot())
}
