[CmdletBinding(DefaultParameterSetName = 'Prepare')]
param(
    [Parameter(ParameterSetName = 'Prepare')][switch]$Prepare,
    [Parameter(Mandatory = $true, ParameterSetName = 'Capture')][string]$Capture,
    [Parameter(ParameterSetName = 'Capture')][switch]$CheckOnly,
    [Parameter(ParameterSetName = 'Capture')][switch]$AllowRemovals
)
$ErrorActionPreference = 'Stop'
# Resolve the caller's relative capture path before changing directories.
$capturePath = if ($Capture) { (Resolve-Path -LiteralPath $Capture).Path }
Push-Location -LiteralPath $PSScriptRoot
try {
    $arguments = @('scripts/update-snapshot.mjs')
    if ($capturePath) {
        $arguments += @('--capture', $capturePath)
        if ($CheckOnly) { $arguments += '--check-only' }
        if ($AllowRemovals) { $arguments += '--allow-removals' }
    } else { $arguments += '--prepare' }
    & node @arguments
    $result = $LASTEXITCODE
    if ($result -eq 0 -and -not $capturePath) {
        try {
            Get-Content -LiteralPath 'JSON/update/capture-snapshot.js' -Raw | Set-Clipboard
            Write-Host 'Capture snippet copied to the clipboard.'
        } catch { Write-Host 'Clipboard unavailable; copy JSON/update/capture-snapshot.js manually.' }
    }
} finally { Pop-Location }
exit $result
