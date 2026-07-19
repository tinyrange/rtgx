//go:build rtg && windows && arm64

package graphics

// rtg:linkstatic comdlg32.dll,GetOpenFileNameW
func windowsGetOpenFileName(data *byte) int { return 0 }

// rtg:linkstatic comdlg32.dll,GetSaveFileNameW
func windowsGetSaveFileName(data *byte) int { return 0 }

// rtg:linkstatic shell32.dll,SHBrowseForFolderW
func windowsBrowseForFolder(data *byte) int { return 0 }

// rtg:linkstatic shell32.dll,SHGetPathFromIDListW
func windowsGetPathFromIDList(item, path int) int { return 0 }

// rtg:linkstatic ole32.dll,CoTaskMemFree
func windowsCoTaskMemFree(memory int) {}
