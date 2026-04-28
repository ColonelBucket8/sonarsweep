$ErrorActionPreference = "Stop"

$repo = "ariffrahimin/sonarsweep"
$binName = "sonarsweep"
$installDir = Join-Path $env:USERPROFILE ".sonarsweep\bin"

Write-Host "==> Installing $binName..."

$os = "windows"
$arch = "amd64"

$downloadUrl = "https://github.com/$repo/releases/latest/download/${binName}-${os}-${arch}.zip"
$tempFile = Join-Path $env:TEMP "${binName}.zip"

Write-Host "==> Downloading from $downloadUrl..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile

Write-Host "==> Extracting..."
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}
Expand-Archive -Path $tempFile -DestinationPath $installDir -Force

Remove-Item -Path $tempFile -Force

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notmatch [regex]::Escape($installDir)) {
    Write-Host "==> Adding $installDir to user PATH..."
    $newPath = $userPath
    if (!$newPath.EndsWith(";")) {
        $newPath += ";"
    }
    $newPath += $installDir
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "==> Please restart your terminal to pick up the new PATH."
}

Write-Host "==> Installation complete! You can now run '$binName'."
