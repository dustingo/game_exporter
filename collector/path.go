package collector

import (
	"path/filepath"
	"strings"
)

//var (
//	rootfsPath = kingpin.Flag("path.rootfs", "rootfs mountpoint.").Default("/").String()
//	procPath   = kingpin.Flag("path.procfs", "procfs mountpoint.").Default("/proc").String()
//)
var (
	rootfsPath = "/"
	procPath   = "/proc"
	sysPath    = "/sys"
)

func rootfsFilePath(name string) string {
	return filepath.Join(rootfsPath, name)
}

func rootfsStripPrefix(path string) string {
	if rootfsPath == "/" {
		return path
	}
	stripped := strings.TrimPrefix(path, rootfsPath)
	if stripped == "" {
		return "/"
	}
	return stripped
}
func procFilePath(name string) string {
	return filepath.Join(procPath, name)
}
