param(
    [string]$CatalogPath = "data/datasets/catalog.json",
    [string]$ComposeService = "postgres",
    [string]$DatabaseUser = "jax",
    [string]$DatabaseName = "jax"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $CatalogPath)) {
    Write-Host "  Dataset catalog not found at $CatalogPath; skipping dataset snapshot sync." -ForegroundColor Yellow
    exit 0
}

$items = Get-Content -LiteralPath $CatalogPath -Raw | ConvertFrom-Json
if ($null -eq $items) {
    $items = @()
} elseif ($items -isnot [array]) {
    $items = @($items)
}
if ($items.Count -eq 0) {
    Write-Host "  Dataset catalog is empty; skipping dataset snapshot sync." -ForegroundColor Yellow
    exit 0
}

function Escape-Sql([object]$Value) {
    if ($null -eq $Value) {
        return ""
    }

    return ([string]$Value).Replace("'", "''")
}

$synced = 0
foreach ($item in $items) {
    if (-not $item.id -or -not $item.hash -or -not $item.name -or -not $item.symbol -or -not $item.file_path) {
        Write-Host "  Skipping incomplete dataset catalog entry: $($item | ConvertTo-Json -Compress)" -ForegroundColor Yellow
        continue
    }

    $recordCount = [int]($item.record_count | Select-Object -First 1)
    $metadata = @{
        importedFrom = $CatalogPath.Replace("\", "/")
        seededBy = "sync-dataset-snapshots.ps1"
        catalogCreatedAt = $item.created_at
    } | ConvertTo-Json -Compress

    $sql = @"
INSERT INTO dataset_snapshots (
    dataset_id,
    dataset_hash,
    dataset_name,
    symbol,
    source,
    schema_ver,
    record_count,
    start_date,
    end_date,
    file_path,
    metadata,
    created_at,
    updated_at,
    last_seen_at
) VALUES (
    '$(Escape-Sql $item.id)',
    '$(Escape-Sql $item.hash)',
    '$(Escape-Sql $item.name)',
    '$(Escape-Sql $item.symbol)',
    '$(Escape-Sql $item.source)',
    '$(Escape-Sql $item.schema_ver)',
    $recordCount,
    '$(Escape-Sql $item.start_date)'::timestamptz,
    '$(Escape-Sql $item.end_date)'::timestamptz,
    '$(Escape-Sql $item.file_path)',
    '$(Escape-Sql $metadata)'::jsonb,
    '$(Escape-Sql $item.created_at)'::timestamptz,
    NOW(),
    NOW()
) ON CONFLICT (dataset_id) DO UPDATE SET
    dataset_hash = EXCLUDED.dataset_hash,
    dataset_name = EXCLUDED.dataset_name,
    symbol = EXCLUDED.symbol,
    source = EXCLUDED.source,
    schema_ver = EXCLUDED.schema_ver,
    record_count = EXCLUDED.record_count,
    start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date,
    file_path = EXCLUDED.file_path,
    metadata = dataset_snapshots.metadata || EXCLUDED.metadata,
    updated_at = NOW(),
    last_seen_at = NOW();
"@

    $sql | docker compose exec -T $ComposeService psql -v ON_ERROR_STOP=1 -U $DatabaseUser -d $DatabaseName | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to sync dataset snapshot '$($item.id)'"
    }
    $synced++
}

Write-Host "  Synced $synced dataset snapshot catalog entries." -ForegroundColor Green
