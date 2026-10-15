# 在 Windows 上生成各个平台的证枢应用。
# 示例：
#   powershell -ExecutionPolicy Bypass -File .\scripts\build-platforms.ps1
#   .\scripts\build-platforms.ps1 -Version 1.2.0 -Platform linux/amd64,windows/amd64
#   .\scripts\build-platforms.cmd -SkipWeb
[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$Output = "dist",
    [string[]]$Platform = @(),
    [switch]$SkipWeb,
    [switch]$SkipZip
)

$ErrorActionPreference = "Stop"

function Write-Utf8NoBom {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Value
    )
    $encoding = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($Path, $Value, $encoding)
}

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)][string]$FileName,
        [Parameter(Mandatory = $true)][string[]]$ArgumentList
    )
    & $FileName @ArgumentList
    if ($LASTEXITCODE -ne 0) {
        throw "$FileName 退出码 $LASTEXITCODE"
    }
}

function Get-RepoRoot {
    $here = Split-Path -Parent $PSCommandPath
    $root = Resolve-Path (Join-Path $here "..")
    if (-not (Test-Path (Join-Path $root "go.mod"))) {
        throw "未在脚本上级目录找到 go.mod"
    }
    return $root.Path
}

function Get-BuildVersion {
    param([string]$Requested, [string]$Root)
    if ($Requested) {
        return $Requested
    }
    $git = Get-Command git -ErrorAction SilentlyContinue
    if ($git) {
        try {
            $described = & git -C $Root describe --tags --always --dirty 2>$null
            if ($LASTEXITCODE -eq 0 -and $described) {
                return $described.Trim()
            }
        }
        catch {
        }
    }
    return "1.0.0"
}

function Convert-Platform {
    param([string]$Text)
    $parts = $Text.Split("/")
    if ($parts.Length -lt 2 -or $parts.Length -gt 3) {
        throw "平台格式应为 os/arch 或 os/arch/goarm，收到: $Text"
    }
    $arm = ""
    if ($parts.Length -eq 3) {
        $arm = $parts[2]
    }
    return @{ OS = $parts[0]; Arch = $parts[1]; Arm = $arm }
}

function Get-Targets {
    param([string[]]$Requested)
    if ($Requested -and $Requested.Count -gt 0) {
        $list = @()
        foreach ($item in $Requested) {
            foreach ($piece in ($item -split ",")) {
                $name = $piece.Trim()
                if ($name) {
                    $list += Convert-Platform $name
                }
            }
        }
        return $list
    }
    return @(
        @{ OS = "linux"; Arch = "amd64"; Arm = "" },
        @{ OS = "linux"; Arch = "arm64"; Arm = "" },
        @{ OS = "linux"; Arch = "arm"; Arm = "7" },
        @{ OS = "windows"; Arch = "amd64"; Arm = "" },
        @{ OS = "windows"; Arch = "arm64"; Arm = "" },
        @{ OS = "darwin"; Arch = "amd64"; Arm = "" },
        @{ OS = "darwin"; Arch = "arm64"; Arm = "" }
    )
}

function Build-Web {
    param([string]$Root)
    $distIndex = Join-Path $Root "internal\web\dist\index.html"
    if ($SkipWeb) {
        if (-not (Test-Path $distIndex)) {
            throw "已指定 -SkipWeb，但缺少 $distIndex"
        }
        Write-Host "跳过前端构建"
        return
    }
    $npm = Get-Command npm -ErrorAction SilentlyContinue
    if (-not $npm) {
        throw "未找到 npm。安装 Node.js 后重试，或在已有前端产物时使用 -SkipWeb"
    }
    Write-Host "正在构建前端"
    Push-Location (Join-Path $Root "web")
    try {
        Invoke-Checked npm @("install")
        Invoke-Checked npm @("run", "build")
    }
    finally {
        Pop-Location
    }
    if (-not (Test-Path $distIndex)) {
        throw "前端构建后仍缺少 $distIndex"
    }
}

function Get-BinaryName {
    param($Target)
    $name = "sslplatform-$($Target.OS)-$($Target.Arch)"
    if ($Target.Arm) {
        $name = "$name" + "v$($Target.Arm)"
    }
    if ($Target.OS -eq "windows") {
        $name += ".exe"
    }
    return $name
}

function New-Package {
    param(
        [string]$Root,
        [string]$BinaryPath,
        [string]$ZipPath,
        $Target
    )
    $stage = Join-Path ([System.IO.Path]::GetTempPath()) ("sslplatform-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path (Join-Path $stage "configs") | Out-Null
    $exeName = "sslplatform"
    if ($Target.OS -eq "windows") {
        $exeName = "sslplatform.exe"
        $launcher = ((@(
                '@echo off',
                'cd /d "%~dp0"',
                'sslplatform.exe'
            ) -join "`r`n") + "`r`n")
        Write-Utf8NoBom (Join-Path $stage "start.cmd") $launcher
    }
    else {
        $launcher = ((@(
                '#!/bin/sh',
                'cd "$(dirname "$0")" || exit 1',
                'chmod +x ./sslplatform',
                'exec ./sslplatform'
            ) -join "`n") + "`n")
        Write-Utf8NoBom (Join-Path $stage "start.sh") $launcher
    }
    Copy-Item $BinaryPath (Join-Path $stage $exeName)
    Copy-Item (Join-Path $Root "configs\config.yaml") (Join-Path $stage "configs\config.yaml")
    if (Test-Path $ZipPath) {
        Remove-Item $ZipPath -Force
    }
    New-ZipFromDirectory -SourceDir $stage -ZipPath $ZipPath
    Remove-Item $stage -Recurse -Force
}

function New-ZipFromDirectory {
    param(
        [Parameter(Mandatory = $true)][string]$SourceDir,
        [Parameter(Mandatory = $true)][string]$ZipPath
    )
    Add-Type -AssemblyName System.IO.Compression | Out-Null
    Add-Type -AssemblyName System.IO.Compression.FileSystem | Out-Null
    $zip = [System.IO.Compression.ZipFile]::Open($ZipPath, [System.IO.Compression.ZipArchiveMode]::Create)
    try {
        $root = (Resolve-Path $SourceDir).Path.TrimEnd('\', '/')
        Get-ChildItem -Path $root -Recurse -File | ForEach-Object {
            $relative = $_.FullName.Substring($root.Length).TrimStart('\', '/')
            $entryName = $relative -replace '\\', '/'
            [void][System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile(
                $zip,
                $_.FullName,
                $entryName,
                [System.IO.Compression.CompressionLevel]::Optimal
            )
        }
    }
    finally {
        $zip.Dispose()
    }
}

$root = Get-RepoRoot
$version = Get-BuildVersion -Requested $Version -Root $root
$targets = @(Get-Targets -Requested $Platform)
if ($targets.Count -eq 0) {
    throw "没有可编译的平台"
}
$outDir = $Output
if (-not [System.IO.Path]::IsPathRooted($outDir)) {
    $outDir = Join-Path $root $outDir
}
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    throw "未找到 go。请安装 Go 1.24 或更高版本并加入 PATH"
}

Build-Web -Root $root

$saved = @{
    CGO_ENABLED = $env:CGO_ENABLED
    GOOS        = $env:GOOS
    GOARCH      = $env:GOARCH
    GOARM       = $env:GOARM
}
$checksums = New-Object System.Collections.Generic.List[string]
try {
    $env:CGO_ENABLED = "0"
    Push-Location $root
    foreach ($target in $targets) {
        if (-not $target.OS -or -not $target.Arch) {
            throw "无法识别平台: $target"
        }
        $binaryName = Get-BinaryName $target
        $binaryPath = Join-Path $outDir $binaryName
        $env:GOOS = $target.OS
        $env:GOARCH = $target.Arch
        if ($target.Arm) {
            $env:GOARM = $target.Arm
        }
        else {
            Remove-Item Env:GOARM -ErrorAction SilentlyContinue
        }
        $ldflags = "-s -w -X sslplatform/internal/appinfo.Version=$version"
        Write-Host "正在编译 $binaryName"
        Invoke-Checked go @("build", "-trimpath", "-ldflags", $ldflags, "-o", $binaryPath, "./cmd/server")
        $hash = (Get-FileHash -Algorithm SHA256 -Path $binaryPath).Hash.ToLowerInvariant()
        $checksums.Add("$hash  $binaryName")
        if (-not $SkipZip) {
            $zipStem = $binaryName
            if ($zipStem.EndsWith(".exe")) {
                $zipStem = $zipStem.Substring(0, $zipStem.Length - 4)
            }
            $zipName = "$zipStem.zip"
            $zipPath = Join-Path $outDir $zipName
            New-Package -Root $root -BinaryPath $binaryPath -ZipPath $zipPath -Target $target
            $zipHash = (Get-FileHash -Algorithm SHA256 -Path $zipPath).Hash.ToLowerInvariant()
            $checksums.Add("$zipHash  $zipName")
        }
    }
}
finally {
    Pop-Location
    foreach ($name in @("CGO_ENABLED", "GOOS", "GOARCH", "GOARM")) {
        $value = $saved[$name]
        if ($null -eq $value) {
            Remove-Item "Env:$name" -ErrorAction SilentlyContinue
        }
        else {
            Set-Item "Env:$name" $value
        }
    }
}

$sumPath = Join-Path $outDir "SHA256SUMS.txt"
Write-Utf8NoBom $sumPath (($checksums -join "`n") + "`n")
Write-Host "已生成到 $outDir"
Write-Host "版本 $version"
