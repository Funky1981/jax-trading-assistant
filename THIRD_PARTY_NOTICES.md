# Jax Third-Party Notice Inventory

This is a machine-supported inventory for the current supported Jax runtime.
It is not a legal opinion and does not replace the licence text or notice
requirements of an eventual distribution.

The complete transitive inventories are generated as ignored CycloneDX JSON
files by `scripts/generate-sbom.ps1`. The frontend inventory is generated from
`frontend/package-lock.json`; Go inventories are generated from every supported
module; the IB bridge inventory is generated from its pinned requirements.

## Direct runtime components

| Component | Version/source | Licence evidence | Upstream / review |
| --- | --- | --- | --- |
| pgx/v5 | 5.9.2 | BSD-3-Clause metadata | https://github.com/jackc/pgx |
| golang-migrate | 4.19.1 | MIT metadata | https://github.com/golang-migrate/migrate |
| golang-jwt/jwt/v5 | 5.2.2 | MIT metadata | https://github.com/golang-jwt/jwt |
| Alpaca Go SDK | 3.3.0 | Apache-2.0 metadata | https://github.com/alpacahq/alpaca-trade-api-go |
| Polygon Go client | 1.16.4 | Machine BOM; verify before redistribution | https://github.com/polygon-io/client-go |
| gofinance/ib | 2019 pseudo-version | LGPLv3 with static-linking exception | https://github.com/gofinance/ib — FORMAL LICENCE REVIEW REQUIRED |
| React / React DOM | 19.x lockfile versions | MIT | https://react.dev |
| MUI / Emotion / Radix / TanStack | lockfile versions | Machine BOM metadata | Upstream links and exact versions are in the frontend BOM |
| ib-insync | 0.9.86 | BSD metadata | https://github.com/erdewit/ib_insync |
| FastAPI | 0.141.1 | MIT metadata | https://github.com/fastapi/fastapi |
| Uvicorn | 0.52.4 | BSD metadata | https://github.com/encode/uvicorn |
| Pydantic / pydantic-settings | 2.13.5 / 2.15.0 | MIT metadata | https://github.com/pydantic/pydantic |
| websockets | 17.1 | BSD-3-Clause metadata | https://github.com/python-websockets/websockets |
| python-dotenv | 1.2.3 | BSD-3-Clause metadata | https://github.com/theskumar/python-dotenv |
| tzdata | 2026.4 | Apache-2.0 metadata | https://github.com/python/tzdata |

## Runtime images

The supported Compose/Dockerfile images are pinned by digest in the source.
PostgreSQL/pgvector, Alpine, nginx, Prometheus, Grafana and Qdrant image-layer
licences must be included or linked by the release packaging process. Grafana
and all bundled image-layer notices require formal review before commercial
redistribution. World Monitor is a separate component and requires its own
notice inventory.

## Notice and licence-text handling

The release process must retain the licence texts/attributions required by
each selected component, especially the LGPL terms for `gofinance/ib` and any
image-layer notices. Do not publish raw private datasets or provider data in a
notice bundle. Re-run the SBOM generator after dependency or image changes.

## Application and data terms

The Jax application licence is `OWNER DECISION REQUIRED`. Software licences
are separate from Alpaca, Polygon, Interactive Brokers, SEC/EDGAR and World
Monitor data/API commercial terms; those CR-02D review flags remain open.
