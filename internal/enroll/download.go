package enroll

import (
	"github.com/KontangoOSS/TangoKore/internal/util"
)

// downloadAndExtractTarGz downloads a .tar.gz and extracts a single named binary.
// Delegates to shared util package.
func downloadAndExtractTarGz(url, destDir, binaryName string) error {
	return util.DownloadAndExtractTarGz(url, destDir, binaryName)
}

// downloadAndExtractZip downloads a .zip and extracts a single named binary.
// Delegates to shared util package.
func downloadAndExtractZip(url, destDir, binaryName string) error {
	return util.DownloadAndExtractZip(url, destDir, binaryName)
}

// downloadBinary downloads a raw binary from url and writes it to destPath with 0755.
// Delegates to shared util package.
func downloadBinary(url, destPath string) error {
	return util.DownloadBinary(url, destPath)
}
