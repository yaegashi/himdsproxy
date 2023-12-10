Set-StrictMode -Version 3
# Set-PSDebug -Trace 1
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$pluginDist = 'https://umsafx00rjmzcrqjpnnc.blob.core.windows.net/c5073b0b-1163-9459-f947-6eead4270c7e/c5073b0b-1163-9459-f947-6eead4270c7e_1.1.0.2.zip'
$pluginName = 'Microsoft.Azure.ActiveDirectory.AADLoginForWindows'
$pluginVersion = '1.1.0.2'

$scriptFolder = Split-Path -Parent -Path $MyInvocation.MyCommand.Definition
$packageFolder = [IO.Path]::Combine($scriptFolder, 'Packages', 'Plugins', $pluginName, $pluginVersion)
$logFolder = [IO.Path]::Combine($scriptFolder, 'Logs', 'Plugins', $pluginName, $pluginVersion)
$eventsFolder = [IO.Path]::Combine($scriptFolder, 'Logs', 'Plugins', $pluginName, 'Events')
$configFolder = [IO.Path]::Combine($packageFolder, 'RuntimeSettings')
$statusFolder = [IO.Path]::Combine($packageFolder, 'Status')
$heartbeatFile = [IO.Path]::Combine($statusFolder, 'HeartBeat.Json')

if (-not (Test-Path -Path $packageFolder)) {
    $null = New-Item -Path $packageFolder -ItemType Directory
    $pluginZipPath = [IO.Path]::Combine($packageFolder, "${pluginName}_${pluginVersion}.zip")
    & curl.exe -s -o $pluginZipPath $pluginDist
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to download plugin: $pluginDist"
    }
    Expand-Archive -Path $pluginZipPath -DestinationPath $packageFolder
}

$null = New-Item -Path $logFolder -ItemType Directory -Force
$null = New-Item -Path $eventsFolder -ItemType Directory -Force
$null = New-Item -Path $configFolder -ItemType Directory -Force
$null = New-Item -Path $statusFolder -ItemType Directory -Force

$handlerEnvironment = @(
    @{
        version            = 1
        handlerEnvironment = @{
            logFolder           = $logFolder
            eventsFolder        = $eventsFolder
            configFolder        = $configFolder
            statusFolder        = $statusFolder
            heartbeatFile       = $heartbeatFile
            deploymentId        = ''
            rolename            = ''
            instance            = ''
            hostResolverAddress = ''
        }
    }
)
$handlerEnvironmentPath = [IO.Path]::Combine($packageFolder, 'HandlerEnvironment.json')
ConvertTo-Json -InputObject $handlerEnvironment | Out-File -FilePath $handlerEnvironmentPath -Encoding ascii

$netshQuery = & netsh interface ipv4 show address 1
if ($LASTEXITCODE -ne 0) {
    throw 'Failed to query interface'
}
if (-not ($netshQuery -match '169.254.169.254')) {
    & netsh interface ipv4 add address 1 169.254.169.254 255.255.0.0
    if ($LASTEXITCODE -ne 0) {
        throw 'Failed to add address'
    }
}

$himdsproxyPath = [IO.Path]::Combine($scriptFolder, 'himdsproxy.exe')
$p0 = Start-Process -FilePath $himdsproxyPath -NoNewWindow -PassThru
Start-Sleep -Seconds 2
if ($p0.HasExited) {
    throw 'Failed to start himdsproxy'
}

try {
    $handlerPath = [IO.Path]::Combine($packageFolder, 'AADLoginForWindowsHandler.exe')

    $p1 = Start-Process -FilePath $handlerPath -ArgumentList install -WorkingDirectory $packageFolder -Wait -NoNewWindow -PassThru
    if ($p1.ExitCode -ne 0) {
        throw 'Failed to install plugin'
    }
    
    $p2 = Start-Process -FilePath $handlerPath -ArgumentList enable -WorkingDirectory $packageFolder -Wait -NoNewWindow -PassThru
    if ($p2.ExitCode -ne 0) {
        throw 'Failed to enable plugin'
    }
}
finally {
    Stop-Process -Id $p0.Id
}
