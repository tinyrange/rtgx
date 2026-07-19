//go:build rtg && windows && 386

package graphics

// rtg:linkstatic comdlg32.dll,GetOpenFileNameA
func windowsGetOpenFileName(data *byte) int { return 0 }

// rtg:linkstatic comdlg32.dll,GetSaveFileNameA
func windowsGetSaveFileName(data *byte) int { return 0 }

// rtg:linkstatic shell32.dll,SHBrowseForFolderA
func windowsBrowseForFolder(data *byte) int { return 0 }

// rtg:linkstatic shell32.dll,SHGetPathFromIDListA
func windowsGetPathFromIDList(item, path int) int { return 0 }

// rtg:linkstatic ole32.dll,CoTaskMemFree
func windowsCoTaskMemFree(memory int) {}
