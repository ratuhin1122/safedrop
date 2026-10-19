param([switch]$Live)
$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
    $defaultJson = docker compose config --format json
    if ($LASTEXITCODE -ne 0) { throw 'Default Compose configuration failed' }
    $default = ($defaultJson -join "`n") | ConvertFrom-Json
    foreach ($name in @('prometheus', 'grafana')) {
        if ($default.services.PSObject.Properties.Name -contains $name) {
            throw "$name enabled without observability profile"
        }
    }
    $profileJson = docker compose --profile observability config --format json
    if ($LASTEXITCODE -ne 0) { throw 'Observability Compose configuration failed' }
    $profile = ($profileJson -join "`n") | ConvertFrom-Json
    foreach ($config in @($default, $profile)) {
        foreach ($name in @('api', 'worker')) {
            if (@($config.services.$name.ports | Where-Object { $_.target -eq 9090 }).Count) {
                throw "$name publishes its metrics listener"
            }
            if ($config.services.$name.environment.METRICS_ADDR -ne ':9090') {
                throw "$name metrics listener differs from scrape target"
            }
        }
    }
    foreach ($name in @('prometheus', 'grafana')) {
        $service = $profile.services.$name
        if ($null -eq $service) { throw "Missing $name service" }
        if (@($service.ports | Where-Object { $_.host_ip -ne '127.0.0.1' }).Count) {
            throw "$name UI is not restricted to localhost"
        }
        if (-not @($service.volumes | Where-Object { $_.type -eq 'bind' -and $_.read_only }).Count) {
            throw "$name lacks a read-only configuration mount"
        }
        if (-not @($service.volumes | Where-Object { $_.type -eq 'volume' }).Count) {
            throw "$name lacks persistent storage"
        }
    }
    if ($Live) {
        $response = Invoke-RestMethod 'http://localhost:9090/api/v1/targets'
        if ($response.status -ne 'success') { throw 'Prometheus targets query failed' }
        foreach ($name in @('api', 'worker')) {
            $targets = @($response.data.activeTargets | Where-Object { $_.labels.job -eq $name })
            if ($targets.Count -ne 1 -or $targets[0].health -ne 'up' -or
                $targets[0].scrapeUrl -ne "http://${name}:9090/metrics") {
                throw "$name scrape target is missing, incorrect, or DOWN"
            }
        }
    }
    Write-Output 'Observability checks passed'
} finally {
    Pop-Location
}
