//go:build rtg && windows && (386 || amd64 || arm64)

package graphics

const windowsDialogBufferBytes = 8192

func windowsReleaseNativeBytes(memory int) {
	if memory != 0 {
		windowsGlobalUnlock(memory)
		windowsGlobalFree(memory)
	}
}

func windowsDialogString(text string) (int, int) {
	if text == "" {
		return 0, 0
	}
	return windowsNativeBytes(windowsNativeString(text))
}

func windowsDialogFile(w *Window, options FileDialogOptions, save bool) (string, bool) {
	if w == nil || w.closed {
		return "", false
	}
	fileData := make([]byte, windowsDialogBufferBytes)
	if options.DefaultName != "" {
		name := windowsNativeString(options.DefaultName)
		copy(fileData, name)
	}
	fileMemory, fileAddress := windowsNativeBytes(fileData)
	if fileMemory == 0 {
		return "", false
	}
	titleMemory, titleAddress := windowsDialogString(options.Title)
	directoryMemory, directoryAddress := windowsDialogString(options.InitialDirectory)
	data := make([]byte, windowsOpenFileNameSize)
	windowsPut32(data, 0, windowsOpenFileNameSize)
	windowsPutPointer(data, windowsOpenFileNameOwnerOffset, w.native)
	windowsPutPointer(data, windowsOpenFileNameFileOffset, fileAddress)
	windowsPut32(data, windowsOpenFileNameMaxFileOffset, len(fileData)/windowsDialogCharacterSize)
	windowsPutPointer(data, windowsOpenFileNameInitialDirectoryOffset, directoryAddress)
	windowsPutPointer(data, windowsOpenFileNameTitleOffset, titleAddress)
	flags := 0x00000008 | 0x00000800 | 0x00080000
	if save {
		flags |= 0x00000002
	} else {
		flags |= 0x00001000
	}
	windowsPut32(data, windowsOpenFileNameFlagsOffset, flags)
	accepted := 0
	if save {
		accepted = windowsGetSaveFileName(&data[0])
	} else {
		accepted = windowsGetOpenFileName(&data[0])
	}
	path := ""
	if accepted != 0 {
		windowsReadMemory(&fileData[0], fileAddress, len(fileData))
		path = windowsClipboardDecode(fileData)
	}
	windowsReleaseNativeBytes(directoryMemory)
	windowsReleaseNativeBytes(titleMemory)
	windowsReleaseNativeBytes(fileMemory)
	return path, accepted != 0 && path != ""
}

func (w *Window) OpenFileDialog(options FileDialogOptions) (string, bool) {
	return windowsDialogFile(w, options, false)
}

func (w *Window) SaveFileDialog(options FileDialogOptions) (string, bool) {
	return windowsDialogFile(w, options, true)
}

func (w *Window) SelectFolderDialog(options FolderDialogOptions) (string, bool) {
	if w == nil || w.closed {
		return "", false
	}
	displayData := make([]byte, windowsDialogBufferBytes)
	displayMemory, displayAddress := windowsNativeBytes(displayData)
	pathData := make([]byte, windowsDialogBufferBytes)
	pathMemory, pathAddress := windowsNativeBytes(pathData)
	titleMemory, titleAddress := windowsDialogString(options.Title)
	if displayMemory == 0 || pathMemory == 0 {
		windowsReleaseNativeBytes(titleMemory)
		windowsReleaseNativeBytes(pathMemory)
		windowsReleaseNativeBytes(displayMemory)
		return "", false
	}
	data := make([]byte, windowsBrowseInfoSize)
	windowsPutPointer(data, windowsBrowseInfoOwnerOffset, w.native)
	windowsPutPointer(data, windowsBrowseInfoDisplayNameOffset, displayAddress)
	windowsPutPointer(data, windowsBrowseInfoTitleOffset, titleAddress)
	// RETURNONLYFSDIRS, EDITBOX, VALIDATE and NEWDIALOGSTYLE retain the classic
	// Shell dialog on old Windows while enabling New Folder on newer shells.
	windowsPut32(data, windowsBrowseInfoFlagsOffset, 0x00000001|0x00000010|0x00000020|0x00000040)
	item := windowsBrowseForFolder(&data[0])
	accepted := item != 0 && windowsGetPathFromIDList(item, pathAddress) != 0
	if item != 0 {
		windowsCoTaskMemFree(item)
	}
	path := ""
	if accepted {
		windowsReadMemory(&pathData[0], pathAddress, len(pathData))
		path = windowsClipboardDecode(pathData)
	}
	windowsReleaseNativeBytes(titleMemory)
	windowsReleaseNativeBytes(pathMemory)
	windowsReleaseNativeBytes(displayMemory)
	return path, accepted && path != ""
}
