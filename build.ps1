param([string]$Data = 'data', [string]$Output = 'dist/uc-print.exe')
Set-Location -LiteralPath $PSScriptRoot
$env:GOCACHE = Join-Path $PSScriptRoot '.go-cache'
$env:GOPROXY = 'off'
$env:GOTOOLCHAIN = 'local'
go run ./tools/ucbuild --data $Data --output $Output
exit $LASTEXITCODE
