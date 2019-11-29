package release

type ReleaseMsgCode int

const (
	ReleasePending              ReleaseMsgCode = 1000
	ReleaseInstallFailed        ReleaseMsgCode = 1001
	ReleaseUpgradeFailed        ReleaseMsgCode = 1002
	ReleaseDeleteFailed         ReleaseMsgCode = 1003
	ReleasePauseOrRecoverFailed ReleaseMsgCode = 1004
	ReleaseFailed               ReleaseMsgCode = 1100

	ReleaseNotReady ReleaseMsgCode = 2000
	ReleasePaused   ReleaseMsgCode = 2001
)
