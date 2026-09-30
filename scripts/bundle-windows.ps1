# Builds the "completo" Windows distribution: cometa.exe + Go toolchain +
# Ebitengine module cache + prewarmed build cache, so games build offline.
#
#   .\scripts\bundle-windows.ps1 -GoVersion 1.26.3 -Version v0.1.0
#
# Output: dist\cometa_<Version>_windows_amd64_completo.zip
param(
	[string]$GoVersion = "1.26.3",
	[string]$Version = "dev",
	[string]$Out = "dist"
)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$stage = Join-Path $root "$Out\stage\cometa"
Remove-Item -Recurse -Force (Join-Path $root "$Out\stage") -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $stage | Out-Null

# 1. Go toolchain (BSD-3-Clause: LICENSE and PATENTS stay inside go\).
$zip = Join-Path $root "$Out\go$GoVersion.zip"
if (-not (Test-Path $zip)) {
	Invoke-WebRequest "https://go.dev/dl/go$GoVersion.windows-amd64.zip" -OutFile $zip
}
Expand-Archive $zip -DestinationPath $stage
foreach ($d in "test", "doc", "misc") { Remove-Item -Recurse -Force (Join-Path $stage "go\$d") -ErrorAction SilentlyContinue }

# 2. The compiler itself.
$env:CGO_ENABLED = "0"
Push-Location $root
$goExe = Join-Path $stage "go\bin\go.exe"
& $goExe build -ldflags "-s -w -X main.version=$Version" -o (Join-Path $stage "cometa.exe") ./cmd/cometa
if ($LASTEXITCODE) { throw "go build failed" }

# 3. Seed the module and build caches by building real games through cometa,
#    so the cache keys match exactly what users will build.
$env:COMETA_HOME = $stage
$env:COMETA_SEMBRAR = "1"
$env:GOFLAGS = ""
foreach ($ex in "09_atrapa", "10_ui", "15_mazmorra", "13_plataformas", "07_sprites") {
	& (Join-Path $stage "cometa.exe") construir "examples\pincel\$ex.cometa" -o (Join-Path $root "$Out\seed.exe")
	if ($LASTEXITCODE) { throw "seed build $ex failed" }
}
Remove-Item (Join-Path $root "$Out\seed.exe") -ErrorAction SilentlyContinue
Remove-Item Env:COMETA_HOME, Env:COMETA_SEMBRAR
Pop-Location

# 4. Licences: Cometa, Go (inside go\), and every module in the cache.
Copy-Item (Join-Path $root "LICENSE"), (Join-Path $root "README.md") $stage
$notices = Join-Path $stage "THIRD_PARTY_NOTICES.txt"
"Third-party software bundled with Cometa`r`n" | Set-Content $notices
"Go toolchain: see go\LICENSE and go\PATENTS (BSD-3-Clause)`r`n" | Add-Content $notices
Get-ChildItem (Join-Path $stage "modcache") -Recurse -File |
	Where-Object { $_.Name -match '^(LICENSE|LICENCE|COPYING|NOTICE)(\.(md|txt))?$' } |
	ForEach-Object {
		$rel = $_.FullName.Substring((Join-Path $stage "modcache").Length + 1)
		"`r`n==== $rel ====`r`n" | Add-Content $notices
		Get-Content $_.FullName | Add-Content $notices
	}

# 5. Zip.
$archive = Join-Path $root "$Out\cometa_${Version}_windows_amd64_completo.zip"
Remove-Item $archive -ErrorAction SilentlyContinue
Compress-Archive -Path $stage -DestinationPath $archive -CompressionLevel Optimal
Write-Host "Listo: $archive"
