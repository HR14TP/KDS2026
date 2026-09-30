Invoke-Expression "del files/*"
New-Item -ItemType Directory -Force -Path "files"

$Env:GOOS = "windows"; $Env:GOARCH = "amd64";
$Env:CGO_ENABLED = "1";
Invoke-Expression "go build -o files/kdspoller-windows .\pkg\."