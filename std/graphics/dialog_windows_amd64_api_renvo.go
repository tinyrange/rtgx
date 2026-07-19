//go:build renvo && windows && amd64

package graphics

// renvo:linkstatic comdlg32.dll,GetOpenFileNameW
func windowsGetOpenFileName(data *byte) int { return 0 }

// renvo:linkstatic comdlg32.dll,GetSaveFileNameW
func windowsGetSaveFileName(data *byte) int { return 0 }

// renvo:linkstatic shell32.dll,SHBrowseForFolderW
func windowsBrowseForFolder(data *byte) int { return 0 }

// renvo:linkstatic shell32.dll,SHGetPathFromIDListW
func windowsGetPathFromIDList(item, path int) int { return 0 }

// renvo:linkstatic ole32.dll,CoTaskMemFree
func windowsCoTaskMemFree(memory int) {}
