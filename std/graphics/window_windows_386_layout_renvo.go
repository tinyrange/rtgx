//go:build renvo && windows && 386

package graphics

const windowsPointerSize = 4

const windowsWindowClassSize = 40
const windowsWindowClassProcOffset = 4
const windowsWindowClassInstanceOffset = 16
const windowsWindowClassCursorOffset = 24
const windowsWindowClassNameOffset = 36

const windowsMessageStructSize = 32
const windowsMessageWindowOffset = 0
const windowsMessageKindOffset = 4
const windowsMessageWParamOffset = 8
const windowsMessageLParamOffset = 12
const windowsMessageTimeOffset = 16
const windowsMessagePointXOffset = 20
const windowsMessagePointYOffset = 24
const windowsMessagePrivateOffset = 28

const windowsTrackMouseEventSize = 16
const windowsTrackMouseEventHoverTimeOffset = 12

const windowsOpenFileNameSize = 76
const windowsOpenFileNameOwnerOffset = 4
const windowsOpenFileNameFileOffset = 28
const windowsOpenFileNameMaxFileOffset = 32
const windowsOpenFileNameInitialDirectoryOffset = 44
const windowsOpenFileNameTitleOffset = 48
const windowsOpenFileNameFlagsOffset = 52

const windowsBrowseInfoSize = 32
const windowsBrowseInfoOwnerOffset = 0
const windowsBrowseInfoDisplayNameOffset = 8
const windowsBrowseInfoTitleOffset = 12
const windowsBrowseInfoFlagsOffset = 16
const windowsDialogCharacterSize = 1
