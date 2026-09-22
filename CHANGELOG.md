# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased


### Features

* **retry:** global token-bucket rate limit (`OO_RATE_LIMIT`/`OO_BURST`), typed
  `TransientError` with `Retry-After`, exponential backoff
  (`OO_RETRY_ATTEMPTS`/`_BASE`/`_MAX`) and a process-wide 429 cooldown gate.
  All HTTP paths are paced via `pacedTransport`.

## [0.19.0](https://github.com/eSlider/go-onlyoffice/compare/v0.18.0...v0.19.0) (2026-09-22)


### ⚠ BREAKING CHANGES

* **cli:** subject-based command tree (tea-style) + global --output flag
* rename oo-cli → oo, split library by domain, relocate applications

### Features

* **auth:** AuthenticateContext + InvalidateToken for long-running syncs ([89d97fb](https://github.com/eSlider/go-onlyoffice/commit/89d97fb8627eb13efa498f866d0c855774322d04))
* **catalog:** mbox header scan, company legal-suffix match, apply order ([036a306](https://github.com/eSlider/go-onlyoffice/commit/036a30654d228d50b27609132bb7b95854bfc4d4))
* **catalog:** port oo catalog (scan/merge/match/apply) from legacy branch ([9a3bf2b](https://github.com/eSlider/go-onlyoffice/commit/9a3bf2b90c3b45e3f8c9c4b6494b8e0098c353ee))
* **catalog:** scan Thunderbird address books and Gloda contacts ([a9d9423](https://github.com/eSlider/go-onlyoffice/commit/a9d9423f6bb23082945a0a20059545e273710986))
* **cli:** subject-based command tree (tea-style) + global --output flag ([606902d](https://github.com/eSlider/go-onlyoffice/commit/606902dab6a3dc96e75530008555ec9fa57504e1))
* **crm:** add invoices CLI and opportunity update ([feebab4](https://github.com/eSlider/go-onlyoffice/commit/feebab4ad1cc929b08bec698cc1e4da062081faf))
* **crm:** add UpdateContactName and CloseCRMTask helpers ([0b192e7](https://github.com/eSlider/go-onlyoffice/commit/0b192e70b817105377b68dea794ed0fe625be204))
* **crm:** allow updating invoice terms via oo ([e5bda10](https://github.com/eSlider/go-onlyoffice/commit/e5bda1031dd2b8c77ab31122c12a80adf762cd34))
* **crm:** contact email & person-opportunity indexes, history entity whitelist ([3184203](https://github.com/eSlider/go-onlyoffice/commit/3184203aaada45218ff9c1d3c05c6b5719c3c172))
* **crm:** dedupe duplicates, fix deal titles, and add cleanup CLI ([6762e9c](https://github.com/eSlider/go-onlyoffice/commit/6762e9cb4ef1fff548fc0d6afeb27d8cb6115f4f))
* **crm:** invoice update notes and PO fields ([3381d64](https://github.com/eSlider/go-onlyoffice/commit/3381d64c6a507fe553c2b24f15d2cdd7cefde2c2))
* **crm:** invoices CLI and opportunity update ([44ffa1c](https://github.com/eSlider/go-onlyoffice/commit/44ffa1c8182cec808c9f99fa81c5f8a8faf4be6e))
* **crm:** merge company slogan variants in dedupe grouping ([10b0759](https://github.com/eSlider/go-onlyoffice/commit/10b075927de1f8cd136390beec6f3da7c23ea1ff))
* **deploy:** rclone WebDAV mount compose + smoke + docs ([#54](https://github.com/eSlider/go-onlyoffice/issues/54)) ([5c959ab](https://github.com/eSlider/go-onlyoffice/commit/5c959ab79cc1dcf63f87e67d63bfbe26ce6dd0db))
* **docs:** Ghostscript PDF optimize for OO ([c7259ae](https://github.com/eSlider/go-onlyoffice/commit/c7259ae8206211a33c5f7d1bfe0017b25221905a))
* **docs:** hOCR → Markdown via go-hocr ([0586f00](https://github.com/eSlider/go-onlyoffice/commit/0586f0007c6fa00a4719d36572d9c6beca62f9eb))
* **docs:** md↔docx convert, OCR→PDF, as-md/put-md for agents ([0907393](https://github.com/eSlider/go-onlyoffice/commit/0907393794ca4fa5b964c91964a5af1dbddb1b2b))
* **docs:** md↔docx, OCR→PDF, as-md/put-md ([5aa2a60](https://github.com/eSlider/go-onlyoffice/commit/5aa2a6046e66df7209080858e56b7400c7bc1ed4))
* **docs:** oo docs hocr — tesseract hOCR → go-hocr Markdown/YAML ([885fa5e](https://github.com/eSlider/go-onlyoffice/commit/885fa5e7d3dbadcdd824774822f3f04b282d48bc))
* **docs:** optimize PDF via Ghostscript pdfwrite ([dc1effa](https://github.com/eSlider/go-onlyoffice/commit/dc1effa8fe30bf4995508f9896a97db23f1f1dc3))
* **docs:** put-txt preserves line breaks in DOCX ([b3ee6d1](https://github.com/eSlider/go-onlyoffice/commit/b3ee6d1cd254f99e5bb8e42b0fb9b29ef424477a))
* **docs:** put-txt with preserved line breaks ([e97cd13](https://github.com/eSlider/go-onlyoffice/commit/e97cd13b9d3d1132dabb8e3c277dd0dd94ad6f3c))
* **docs:** put-xlsx with excelize cutover workbook ([aa15707](https://github.com/eSlider/go-onlyoffice/commit/aa157072e7f4bbb10989f4e70344d2ebcae8d065))
* **files:** add WebDAV-oriented Files operations ([b827fa9](https://github.com/eSlider/go-onlyoffice/commit/b827fa928dd935d48a35f89979fb49915e03bb55))
* **files:** canonical Entry + FileStore REST/DAV adapters ([#35](https://github.com/eSlider/go-onlyoffice/issues/35)) ([7bfdb8f](https://github.com/eSlider/go-onlyoffice/commit/7bfdb8f405fafe6ea23a66cd64c4004ef152b699))
* **files:** dedupe by stem|ext + oo projects files dedupe ([b8822ef](https://github.com/eSlider/go-onlyoffice/commit/b8822ef8d7571df1cae05d2dec85121fcc97a710))
* **files:** dedupe command + stem|ext matching ([ae8b4ed](https://github.com/eSlider/go-onlyoffice/commit/ae8b4ed2374e190bc897dcc02b8889101009c1cf))
* **files:** Documents Dav ops + UpdateFile, fileops errors ([#152](https://github.com/eSlider/go-onlyoffice/issues/152)) ([943eee4](https://github.com/eSlider/go-onlyoffice/commit/943eee42120643d714603cb108a8b2f5568f5c45))
* **files:** MinIO fallback for stale S3 downloads ([#152](https://github.com/eSlider/go-onlyoffice/issues/152)) ([8ee3bbe](https://github.com/eSlider/go-onlyoffice/commit/8ee3bbe119ec124187afac6ee6894718de153e04))
* **files:** project/task Documents API + oo projects|tasks files ([d6c4117](https://github.com/eSlider/go-onlyoffice/commit/d6c41178647fb6640b0352af6d747e16a7cc20b2))
* **files:** read-only SQL file store over Community Server DB ([#36](https://github.com/eSlider/go-onlyoffice/issues/36)) ([77c569e](https://github.com/eSlider/go-onlyoffice/commit/77c569e9885940e0feb0417a3e3c213811da74fb))
* **files:** SQL file backend via Client.FileStore/SQLFileStore + live MySQL integration ([#55](https://github.com/eSlider/go-onlyoffice/issues/55)) ([9f13537](https://github.com/eSlider/go-onlyoffice/commit/9f13537e118f73af0692d0f97a1cc3f47a5fb458))
* **mail:** draft, attach, and draft-invoice CLI ([27c1eca](https://github.com/eSlider/go-onlyoffice/commit/27c1ecad7cda3e128a824300d36998f96ae0a4b2))
* **mail:** oo mails send — SendMail + guard empty-by-id send ([5bbd36c](https://github.com/eSlider/go-onlyoffice/commit/5bbd36ce775962fe266545da0f686b0c3af181bb))
* **mail:** oo mails send — SendMail client + guard empty-by-id ([bb62599](https://github.com/eSlider/go-onlyoffice/commit/bb62599aa8e3c9480be6455af498bf2340308aa8))
* **mails:** add Workspace mail CLI with pagination and parsed from fields ([eb62c53](https://github.com/eSlider/go-onlyoffice/commit/eb62c537f6d222834202b4b29892519c92feea2e))
* **mailsync:** FetchMailFolder — integration-layer walk for ETL consumers ([f45064b](https://github.com/eSlider/go-onlyoffice/commit/f45064b07642e7b53643e33a6e64940106a032fe))
* merge oo-cli and extend library with Calendar/CRM/subtasks/files ([c037233](https://github.com/eSlider/go-onlyoffice/commit/c0372334ee08c9ac1c5722a279412b73ec61dc3e))
* **office:** add Workspace TUI with shared bootstrap and test suite ([be9b54e](https://github.com/eSlider/go-onlyoffice/commit/be9b54edad5af29a21ca67e5d311497b9eb24776))
* **office:** mail preview, infinite scroll, pane resize, and scrollbars ([e550271](https://github.com/eSlider/go-onlyoffice/commit/e55027102efb7e6c7e3401c6b5235d0f0a84bdc6))
* **office:** scrollable panes, nav tree drill-down, and item actions ([82699a9](https://github.com/eSlider/go-onlyoffice/commit/82699a977ef8345636ea8a863525721c36709a74))
* **office:** table detail panes and Alt+1/2/3 layout toggles ([e61498b](https://github.com/eSlider/go-onlyoffice/commit/e61498b97e62073fe4663223fcca4f5da4186cd4))
* **office:** users admin UI, table rendering, and user save fixes ([15d8a29](https://github.com/eSlider/go-onlyoffice/commit/15d8a29bebfcab13b3ba1c8f4358c5feae2aac22))
* **oo:** catalog scan/match/apply for CRM clients and contacts ([b829220](https://github.com/eSlider/go-onlyoffice/commit/b829220ef465f493b426d7eaf841042ffcebaf2e))
* **oo:** catalog scan/match/apply for CRM clients and contacts ([9a93e9e](https://github.com/eSlider/go-onlyoffice/commit/9a93e9e767d6b9c1c9b18a6e30e6c1ba84ed2814))
* **oo:** dav rm — удаление папок/файлов Documents ([#30](https://github.com/eSlider/go-onlyoffice/issues/30)) ([88c4ea4](https://github.com/eSlider/go-onlyoffice/commit/88c4ea473efb96ae486efefca0962992e7f0993c))
* **oo:** docs pdf --stream/--pipe (converted bytes to stdout) ([2476464](https://github.com/eSlider/go-onlyoffice/commit/24764646fcd33c13821f72644f5091cfa1306b29))
* **oo:** file deep links, login check, replace-in (fresh upload) ([2c00101](https://github.com/eSlider/go-onlyoffice/commit/2c0010186167a4a63dfcbcddaf34ee69990504c5))
* **oo:** kontoblatt bulk tools, oo dav, update, retry ([#22](https://github.com/eSlider/go-onlyoffice/issues/22)) ([55f6e3d](https://github.com/eSlider/go-onlyoffice/commit/55f6e3dce4f53e1e62ff1b1253fe2410a4e94ff5))
* **oo:** native document conversion (docs pdf/presigned) ([fc1df27](https://github.com/eSlider/go-onlyoffice/commit/fc1df27f3ace50f2d1d9403b9f5f692a550da66b))
* **oo:** project team CRUD and user lifecycle (block/unblock/password/delete) ([73ddecc](https://github.com/eSlider/go-onlyoffice/commit/73ddecca33b1c149c2ef3b2f60c6838cddc90ea9))
* **oo:** projects files update (overwrite existing file content) ([44ecaff](https://github.com/eSlider/go-onlyoffice/commit/44ecaff89cb7668943faed71b296c878727f630f))
* **oo:** projects milestone-delete ([f224085](https://github.com/eSlider/go-onlyoffice/commit/f2240858108974a3dcd7a5bbb86c0294ffad4cbb))
* **oo:** sheet-aware spreadsheet export (docs csv/json) ([506d717](https://github.com/eSlider/go-onlyoffice/commit/506d717d5f82de6b3560a70545fd690780ee3453))
* **projects:** link CRM companies and git authors to projects ([875467c](https://github.com/eSlider/go-onlyoffice/commit/875467cfe3f5e17918943e98ed43e447b167058e))
* **search:** Elasticsearch searcher (name+content) and oo search ([#37](https://github.com/eSlider/go-onlyoffice/issues/37)) ([dd27a41](https://github.com/eSlider/go-onlyoffice/commit/dd27a41bbd586b8075a37230f71910d6f0446492))
* **search:** full unique path first + immediate folder in results ([#51](https://github.com/eSlider/go-onlyoffice/issues/51)) ([c6c0a02](https://github.com/eSlider/go-onlyoffice/commit/c6c0a025f795cd64ca21a9b891df34fde32b680c))
* **search:** index embedded PDF attachment text ([#42](https://github.com/eSlider/go-onlyoffice/issues/42)) ([d183995](https://github.com/eSlider/go-onlyoffice/commit/d183995647dad07a937ede767c34e79a15c84614))
* **search:** PDF content via own ES index and oo index ([#42](https://github.com/eSlider/go-onlyoffice/issues/42)) ([29c5224](https://github.com/eSlider/go-onlyoffice/commit/29c522490ec392dce7dbc84d4a56fef60f548b7a))
* **search:** substring/AND terms, nested folder scope, limit 1000 ([#49](https://github.com/eSlider/go-onlyoffice/issues/49)) ([6e43144](https://github.com/eSlider/go-onlyoffice/commit/6e43144bf28e1da25832d3babaec3f052d2abccb))
* **security:** secret-scan via gitleaks in CI + pre-push/pre-commit hooks ([#142](https://github.com/eSlider/go-onlyoffice/issues/142)) ([1f6ec1d](https://github.com/eSlider/go-onlyoffice/commit/1f6ec1deb49087d499d9da0b40622836cae83d29))
* upstream generic workspace tooling (board-sync, crm audit, catalog names) ([afa5ceb](https://github.com/eSlider/go-onlyoffice/commit/afa5cebcbed81f294b24c868beba5ea8f9609bd4))


### Bug Fixes

* **ci:** dispatch GoReleaser after release-please; allow Tests dispatch ([#13](https://github.com/eSlider/go-onlyoffice/issues/13)) ([f88bbec](https://github.com/eSlider/go-onlyoffice/commit/f88bbec147a33859e7056cf0ae635bf2ee5cb89b))
* **ci:** gitleaks через бинарник на $GITHUB_WORKSPACE (Gitea runner) ([#10](https://github.com/eSlider/go-onlyoffice/issues/10)) ([c0b2998](https://github.com/eSlider/go-onlyoffice/commit/c0b299877f11cc8153de232fcea9b99266147f35))
* **ci:** point gitleaks at /github/workspace in docker action ([24b89e6](https://github.com/eSlider/go-onlyoffice/commit/24b89e6ab6e27411c8824bf446d42639b37c2a88))
* **ci:** retry tag fetch in Release workflow ([#11](https://github.com/eSlider/go-onlyoffice/issues/11)) ([1ee4fb6](https://github.com/eSlider/go-onlyoffice/commit/1ee4fb687bd1f852c648cc1c64662ce92a439ce0))
* **crm:** clean person names; canonical project titles ([54104e7](https://github.com/eSlider/go-onlyoffice/commit/54104e705ce44656162617358d15e4c132fbdd34))
* **crm:** deterministic sortBy=id in contact paged lists ([f4af808](https://github.com/eSlider/go-onlyoffice/commit/f4af80856c14e2d94708eab4e1b2802bac68ca02))
* **crm:** JSON person update and oo persons update CLI ([2508639](https://github.com/eSlider/go-onlyoffice/commit/250863985c6ac0d2af6ee4600bbcc087901db86b))
* **crm:** link invoices to opportunities ([897cc67](https://github.com/eSlider/go-onlyoffice/commit/897cc67e8ad4e1cd18882276c1e80d3b6083e94e))
* **crm:** store contact addresses via ContactInfo Address type ([#288](https://github.com/eSlider/go-onlyoffice/issues/288)) ([198479a](https://github.com/eSlider/go-onlyoffice/commit/198479a088ca8e0a0f4b8226d8cbf43b81cc108b))
* **docpipe:** index .yaml and extensionless PDF attachments ([#47](https://github.com/eSlider/go-onlyoffice/issues/47)) ([16ef46b](https://github.com/eSlider/go-onlyoffice/commit/16ef46be9d635af11668adb66e3f5ac5f7be16f7))
* **docs:** fixed-width txt as monospace code block ([b2dee29](https://github.com/eSlider/go-onlyoffice/commit/b2dee299707f2395ff37f6c3f336e812b2194d76))
* **docs:** put-md upsert — no duplicate folder files ([c4872f1](https://github.com/eSlider/go-onlyoffice/commit/c4872f10bcd55e198a0d7d3689d59ea2e604d716))
* **docs:** put-md upsert by stem to avoid duplicate folder files ([736bcb8](https://github.com/eSlider/go-onlyoffice/commit/736bcb84f13190c5f1c954bc908d24ee53c0b0b3))
* **docs:** put-txt uses code block for fixed-width extracts ([b00432c](https://github.com/eSlider/go-onlyoffice/commit/b00432c137c8d05dff662a62246f7095e51d4f36))
* **files:** dedupe ProviderPG after facade/SQL merge, rename test fake ([89d018c](https://github.com/eSlider/go-onlyoffice/commit/89d018c1c93019a821ab5b3b469b0eb29c817982))
* **files:** DeleteDavItems with Immediately true ([77b5cb8](https://github.com/eSlider/go-onlyoffice/commit/77b5cb8cfc535cdf93a3f19f7570baae440a6a26))
* **files:** DeleteFiles actually removes files on example OO ([2b873ab](https://github.com/eSlider/go-onlyoffice/commit/2b873abf6c676289af4f29da0d96273520ddd615))
* **files:** DeleteFiles via per-file DELETE API ([612839c](https://github.com/eSlider/go-onlyoffice/commit/612839cd3776649f9d11838d9bc65ab1448402fb))
* **files:** permanent delete via DeleteDavItems ([2bbe696](https://github.com/eSlider/go-onlyoffice/commit/2bbe6964c4642627321fccd7c23757207dc0793c))
* **files:** REST FileStore resolves folders for stat/rename/move/delete ([#62](https://github.com/eSlider/go-onlyoffice/issues/62)) ([c60f59e](https://github.com/eSlider/go-onlyoffice/commit/c60f59efe57970115dc62ab87e5f679906ebc7d1))
* **files:** rewrite viewUrl host to API base on download ([cda1082](https://github.com/eSlider/go-onlyoffice/commit/cda108238c1e03600f9c39296494a1ee04590e43))
* **files:** UpdateFile uses PUT /api/2.0/files/{id}/update ([#25](https://github.com/eSlider/go-onlyoffice/issues/25)) ([6bff5ab](https://github.com/eSlider/go-onlyoffice/commit/6bff5ab7cb470c483494a046a735a3a554cf6345))
* **files:** upsert uploads by default and dedupe project root ([6c9e927](https://github.com/eSlider/go-onlyoffice/commit/6c9e9275c1397608c8b470a0ef565badd1c1396d))
* **files:** пропускать пустой dedup-ключ (dotfiles) ([#63](https://github.com/eSlider/go-onlyoffice/issues/63)) ([22f1482](https://github.com/eSlider/go-onlyoffice/commit/22f14826a9d138269ae58b514822adee25e440c8))
* **files:** ретраить transient-ответы при удалении файлов ([#63](https://github.com/eSlider/go-onlyoffice/issues/63)) ([0c17b91](https://github.com/eSlider/go-onlyoffice/commit/0c17b91ccd75e4eb4e2e9e54f1edbf3ee675935d))
* **mail:** German spacing for invoice draft template ([f132924](https://github.com/eSlider/go-onlyoffice/commit/f132924b44a5d5b0b86c8635a7920dc7e0f84a2e))
* **oo:** assign owner and deadline on task create ([#9](https://github.com/eSlider/go-onlyoffice/issues/9)) ([72f944d](https://github.com/eSlider/go-onlyoffice/commit/72f944dd7cf6f152a6cab9a4014835ceea040546))
* **oo:** skip junk dirs in applications Discover ([cac168b](https://github.com/eSlider/go-onlyoffice/commit/cac168b5889e60ffe3cfbc6242dacc46358c48bf))
* **oo:** skip junk dirs in applications Discover ([abd3e5d](https://github.com/eSlider/go-onlyoffice/commit/abd3e5d8fe8fbb09cdb0cf00bc0496c5546838b6))
* **pdfamount:** widen amount labels and formats ([#27](https://github.com/eSlider/go-onlyoffice/issues/27)) ([01bc055](https://github.com/eSlider/go-onlyoffice/commit/01bc05530be2fe2bd08a9eb585b25f82f0727f80))
* **pdfamount:** не считать ставку НДС суммой; итог DKV ([#29](https://github.com/eSlider/go-onlyoffice/issues/29)) ([7483c03](https://github.com/eSlider/go-onlyoffice/commit/7483c03fbd5064ff4447039855daabfef1e46b9e))
* **retry:** retry transient edge answers centrally, fix stale task test ([#57](https://github.com/eSlider/go-onlyoffice/issues/57)) ([682597c](https://github.com/eSlider/go-onlyoffice/commit/682597c259f353c448c53933bf0f1e4da6ec30c0))
* **retry:** глобальный rate-limit + Retry-After + cooldown ([#70](https://github.com/eSlider/go-onlyoffice/issues/70)) ([295b8dc](https://github.com/eSlider/go-onlyoffice/commit/295b8dc60390ed6c31c9d360329b6ae98e71e860))


### Code Refactoring

* DRY auth path, fix Sprintf/RE2 bugs; real integration tests only ([5ebbb96](https://github.com/eSlider/go-onlyoffice/commit/5ebbb96140c98fc576d71efc1f592cf59ae97083))
* **files:** single file client facade + CLI/TUI migration ([#38](https://github.com/eSlider/go-onlyoffice/issues/38)) ([63e82cb](https://github.com/eSlider/go-onlyoffice/commit/63e82cb6e58c22a5dcbee8b1aeeac9cadddc6cc7))
* **office:** generalize DataTable layout and document TUI table skill ([e427c6d](https://github.com/eSlider/go-onlyoffice/commit/e427c6d6b7be92e423aaf7c52e5c021a96231dbb))
* rename oo-cli → oo, split library by domain, relocate applications ([9c81b61](https://github.com/eSlider/go-onlyoffice/commit/9c81b61a82e4498549b4caa854e6e96ce528f49d))
* **search:** use canonical model from file_core.go ([#37](https://github.com/eSlider/go-onlyoffice/issues/37)) ([520a2c8](https://github.com/eSlider/go-onlyoffice/commit/520a2c8bb44c30f76714c731b8146343eeaabaab))


### Documentation

* add library examples for calendar, crm, subtasks, applications ([c3dba87](https://github.com/eSlider/go-onlyoffice/commit/c3dba87d804038353e07ec4e0bd5ff50f7201e73))
* CHANGELOG 0.6.0, README, AGENTS.md, cmd/oo/main.go tree. ([d6c4117](https://github.com/eSlider/go-onlyoffice/commit/d6c41178647fb6640b0352af6d747e16a7cc20b2))
* CHANGELOG for office v0.5.1 release ([c203dfd](https://github.com/eSlider/go-onlyoffice/commit/c203dfd038b1eee60b7ad0584e657a4251efbd53))
* **crm:** capture association graph and invoice/mail quirks ([cc7df4b](https://github.com/eSlider/go-onlyoffice/commit/cc7df4b0139bec5d988b44c58f6e530046c770d4))
* **crm:** clarify mail API send vs signature for chat links ([20aadfa](https://github.com/eSlider/go-onlyoffice/commit/20aadfad47435c41de4725dd74721ad6a84dfaab))
* **crm:** Team vs Contacts and persons update note ([9237f83](https://github.com/eSlider/go-onlyoffice/commit/9237f835c5bde7413ff0ccd787ade984e215e3c2))
* **files:** add unified file client contract ([#39](https://github.com/eSlider/go-onlyoffice/issues/39)) ([420cffa](https://github.com/eSlider/go-onlyoffice/commit/420cffae92640409d56f85df4846d18eac1c778d))
* **funding:** eSlider support links (reverse-import GitHub f6ecb9b) ([ad4c91c](https://github.com/eSlider/go-onlyoffice/commit/ad4c91c24c50ded1a94af45c2a44c54cf30823e1))
* **oo:** comment where the new helpers are used ([7fbf897](https://github.com/eSlider/go-onlyoffice/commit/7fbf8972b51f6c20ef52f1447e666c753af40789))
* **oo:** dav, documents files api, bulk tools, fix verbs ([#22](https://github.com/eSlider/go-onlyoffice/issues/22)) ([e1af954](https://github.com/eSlider/go-onlyoffice/commit/e1af954afb4614c3c02b569cb8699d2b75ee505c))
* **readme:** document oo docs hocr ([20edad3](https://github.com/eSlider/go-onlyoffice/commit/20edad313eca6042c2e8962641bf4c17b2c22c97))
* **readme:** document oo docs md↔docx / OCR agent workflow ([ea06c4b](https://github.com/eSlider/go-onlyoffice/commit/ea06c4ba0d3f6038ee115a8cebe79faf3758e991))
* **readme:** examples for links, conversion, team/users CRUD ([f5330ca](https://github.com/eSlider/go-onlyoffice/commit/f5330cab970a036b5aa71c5f4aae52e74b39886d))
* Testing section, rclone/SQL refs, docs index ([#53](https://github.com/eSlider/go-onlyoffice/issues/53)) ([c5f089a](https://github.com/eSlider/go-onlyoffice/commit/c5f089a949d869e47f0efa2ec55e56f095829a4c))
* карта поиска и обновления индексов ([#34](https://github.com/eSlider/go-onlyoffice/issues/34)) ([931872d](https://github.com/eSlider/go-onlyoffice/commit/931872d54414e236a8610f8f65bc65537aea3f5a))
* убрать потребительские детали match из index-and-search ([#34](https://github.com/eSlider/go-onlyoffice/issues/34)) ([792f15c](https://github.com/eSlider/go-onlyoffice/commit/792f15c5c26866f499551c7fd551fe4333d571f6))

## [0.18.0](https://github.com/eSlider/go-onlyoffice/compare/v0.17.0...v0.18.0) (2026-09-04)


### Features

* **crm:** add UpdateContactName and CloseCRMTask helpers ([4d8af7a](https://github.com/eSlider/go-onlyoffice/commit/4d8af7a2fef1fcfd96ad913cefc6497e3389963b))


### Bug Fixes

* **crm:** deterministic sortBy=id in contact paged lists ([4d91726](https://github.com/eSlider/go-onlyoffice/commit/4d917261792203732ac739644c9c82124b2166eb))


### Documentation

* **funding:** eSlider support links (reverse-import GitHub e9c969a) ([d650a16](https://github.com/eSlider/go-onlyoffice/commit/d650a16a36037949505eb017c92bd3283e6e2f0f))

## [0.17.0](https://github.com/eSlider/go-onlyoffice/compare/v0.16.0...v0.17.0) (2026-08-31)


### Features

* **mailsync:** FetchMailFolder — integration-layer walk for ETL consumers ([35f0cb8](https://github.com/eSlider/go-onlyoffice/commit/35f0cb8d20076244141065c07e203e633bc3612a))


### Bug Fixes

* **files:** upsert uploads by default and dedupe project root ([24ca144](https://github.com/eSlider/go-onlyoffice/commit/24ca144b22abc5d056a5fd1ed9a1887f26a79d15))

## [0.16.0](https://github.com/eSlider/go-onlyoffice/compare/v0.15.0...v0.16.0) (2026-08-30)

### Features

* **docs:** `put-xlsx` — multi-sheet бюджеты с named inputs, SUM/AVG/MIN
  формулами, cross-sheet ссылками и cell comments (`internal/xlspipe`,
  excelize) ([68445b0](https://github.com/eSlider/go-onlyoffice/commit/68445b0))

### Added

* **docs:** CRM association graph and OO quirks (`docs/crm-associations.md`)
* **crm:** `ForceRegenerateInvoicePDF`, `SetInvoiceStatus`, `PurgeStaleInvoicePDFs`, contact/opportunity file list helpers
* **oo:** `invoices pdf`, `pdf-cleanup`, `status`; create `--consignee`; draft-invoice force-regens PDF

### Fixed

* Document that invoice→deal must be set at create (`update --opportunity` often HTTP 400)

## [0.15.0](https://github.com/eSlider/go-onlyoffice/compare/v0.14.0...v0.15.0) (2026-08-29)

### Features

* **files:** `ListFolder`, `CreateFolder`, `MoveFiles`, `UploadToFolder` — Documents folder helpers for OO Documents ingestion ([6ea2fba](https://github.com/eSlider/go-onlyoffice/commit/6ea2fba))
* **files:** dedupe by stem|ext (`files_stem.go`) + `oo projects files dedupe` ([ac06d86](https://github.com/eSlider/go-onlyoffice/commit/ac06d86))
* **docs:** `internal/docpipe` — md↔docx convert, OCR→PDF, hOCR→Markdown via go-hocr, as-md/put-md for agents ([6ea2fba](https://github.com/eSlider/go-onlyoffice/commit/6ea2fba), [a264cbd](https://github.com/eSlider/go-onlyoffice/commit/a264cbd))
* **docs:** optimize PDF via Ghostscript pdfwrite ([6b3f40e](https://github.com/eSlider/go-onlyoffice/commit/6b3f40e))
* **security:** gitleaks secret-scan in CI + pre-push/pre-commit hooks ([9ead554](https://github.com/eSlider/go-onlyoffice/commit/9ead554))

### Fixes

* **files:** `DeleteFiles` via per-file DELETE API; `DeleteDavItems` with `Immediately` true ([622dcdb](https://github.com/eSlider/go-onlyoffice/commit/622dcdb), [d9a7adc](https://github.com/eSlider/go-onlyoffice/commit/d9a7adc))
* **docs:** put-txt preserves line breaks in DOCX; fixed-width extracts in code block; put-md upsert by stem ([309ae44](https://github.com/eSlider/go-onlyoffice/commit/309ae44), [f1739dc](https://github.com/eSlider/go-onlyoffice/commit/f1739dc), [50bd475](https://github.com/eSlider/go-onlyoffice/commit/50bd475))
* **ci:** point gitleaks at `/github/workspace` in docker action ([2c0df0d](https://github.com/eSlider/go-onlyoffice/commit/2c0df0d))

## [0.14.0](https://github.com/eSlider/go-onlyoffice/compare/v0.13.0...v0.14.0) (2026-08-27)


### Features

* **crm:** contact email & person-opportunity indexes, history entity whitelist ([9c450a0](https://github.com/eSlider/go-onlyoffice/commit/9c450a0352c25f12ac16d91062744ab026ffe660))
* **docs:** hOCR → Markdown via go-hocr ([e531de2](https://github.com/eSlider/go-onlyoffice/commit/e531de280f2c521a7411f75212803649b177db58))
* **docs:** md↔docx convert, OCR→PDF, as-md/put-md for agents ([6ea2fba](https://github.com/eSlider/go-onlyoffice/commit/6ea2fbadf768d7d85f6c7a49a9bc1050e642bc56))
* **docs:** md↔docx, OCR→PDF, as-md/put-md ([db9ef12](https://github.com/eSlider/go-onlyoffice/commit/db9ef12ff4096f8aadb557ca128b6edcb503028c))
* **docs:** oo docs hocr — tesseract hOCR → go-hocr Markdown/YAML ([a264cbd](https://github.com/eSlider/go-onlyoffice/commit/a264cbd61c12be9098c8d16e7c1c1253eb460816))
* **mail:** oo mails send — SendMail + guard empty-by-id send ([dda2bd3](https://github.com/eSlider/go-onlyoffice/commit/dda2bd3ca5cea6ebb89e1ac02b785b9838727bda))
* **mail:** oo mails send — SendMail client + guard empty-by-id ([ab7dfbd](https://github.com/eSlider/go-onlyoffice/commit/ab7dfbd8594de5724286e2b460e789f4acb114bc))
* **security:** secret-scan via gitleaks in CI + pre-push/pre-commit hooks ([#142](https://github.com/eSlider/go-onlyoffice/issues/142)) ([9ead554](https://github.com/eSlider/go-onlyoffice/commit/9ead554f5cc229687d3a27934a023fb1dddaef15))


### Bug Fixes

* **ci:** point gitleaks at /github/workspace in docker action ([2c0df0d](https://github.com/eSlider/go-onlyoffice/commit/2c0df0d55fb03555b60481695554d75dceffce48))
* **docs:** put-md upsert — no duplicate folder files ([2b267d3](https://github.com/eSlider/go-onlyoffice/commit/2b267d36a8086b14a94706468f7e92e9179e62d4))
* **docs:** put-md upsert by stem to avoid duplicate folder files ([50bd475](https://github.com/eSlider/go-onlyoffice/commit/50bd47570b6430cebde43641a7b1d2f8d54d32f2))
* **files:** DeleteFiles actually removes files on example OO ([a8cb4e9](https://github.com/eSlider/go-onlyoffice/commit/a8cb4e97805226e8af694ec6a64ee3ac6a5e9fdd))
* **files:** DeleteFiles via per-file DELETE API ([622dcdb](https://github.com/eSlider/go-onlyoffice/commit/622dcdb7bffd49acbea5ef07f5d5d009f44b1358))


### Documentation

* **readme:** document oo docs hocr ([70e605b](https://github.com/eSlider/go-onlyoffice/commit/70e605b4ca8ee005e4351b2fc2d2eec081604998))
* **readme:** document oo docs md↔docx / OCR agent workflow ([239c5ea](https://github.com/eSlider/go-onlyoffice/commit/239c5ea67201e62de6ba067f0e7963c7c9bda188))

## [0.13.0](https://github.com/eSlider/go-onlyoffice/compare/v0.12.0...v0.13.0) (2026-08-27)


### Features

* **crm:** contact email & person-opportunity indexes, history entity whitelist ([9c450a0](https://github.com/eSlider/go-onlyoffice/commit/9c450a0352c25f12ac16d91062744ab026ffe660))
* **docs:** hOCR → Markdown via go-hocr ([e531de2](https://github.com/eSlider/go-onlyoffice/commit/e531de280f2c521a7411f75212803649b177db58))
* **docs:** md↔docx convert, OCR→PDF, as-md/put-md for agents ([6ea2fba](https://github.com/eSlider/go-onlyoffice/commit/6ea2fbadf768d7d85f6c7a49a9bc1050e642bc56))
* **docs:** md↔docx, OCR→PDF, as-md/put-md ([db9ef12](https://github.com/eSlider/go-onlyoffice/commit/db9ef12ff4096f8aadb557ca128b6edcb503028c))
* **docs:** oo docs hocr — tesseract hOCR → go-hocr Markdown/YAML ([a264cbd](https://github.com/eSlider/go-onlyoffice/commit/a264cbd61c12be9098c8d16e7c1c1253eb460816))
* **mail:** oo mails send — SendMail + guard empty-by-id send ([dda2bd3](https://github.com/eSlider/go-onlyoffice/commit/dda2bd3ca5cea6ebb89e1ac02b785b9838727bda))
* **mail:** oo mails send — SendMail client + guard empty-by-id ([ab7dfbd](https://github.com/eSlider/go-onlyoffice/commit/ab7dfbd8594de5724286e2b460e789f4acb114bc))
* **security:** secret-scan via gitleaks in CI + pre-push/pre-commit hooks ([#142](https://github.com/eSlider/go-onlyoffice/issues/142)) ([9ead554](https://github.com/eSlider/go-onlyoffice/commit/9ead554f5cc229687d3a27934a023fb1dddaef15))


### Bug Fixes

* **ci:** point gitleaks at /github/workspace in docker action ([2c0df0d](https://github.com/eSlider/go-onlyoffice/commit/2c0df0d55fb03555b60481695554d75dceffce48))
* **files:** rewrite viewUrl host to API base on download ([ecf34ba](https://github.com/eSlider/go-onlyoffice/commit/ecf34ba51aae77f1626a60795f7329f25d9344e5))


### Documentation

* **readme:** document oo docs hocr ([70e605b](https://github.com/eSlider/go-onlyoffice/commit/70e605b4ca8ee005e4351b2fc2d2eec081604998))
* **readme:** document oo docs md↔docx / OCR agent workflow ([239c5ea](https://github.com/eSlider/go-onlyoffice/commit/239c5ea67201e62de6ba067f0e7963c7c9bda188))

## [0.12.0](https://github.com/eSlider/go-onlyoffice/compare/v0.11.0...v0.12.0) (2026-08-27)


### Features

* **crm:** contact email & person-opportunity indexes, history entity whitelist ([9c450a0](https://github.com/eSlider/go-onlyoffice/commit/9c450a0352c25f12ac16d91062744ab026ffe660))
* **docs:** md↔docx convert, OCR→PDF, as-md/put-md for agents ([6ea2fba](https://github.com/eSlider/go-onlyoffice/commit/6ea2fbadf768d7d85f6c7a49a9bc1050e642bc56))
* **docs:** md↔docx, OCR→PDF, as-md/put-md ([db9ef12](https://github.com/eSlider/go-onlyoffice/commit/db9ef12ff4096f8aadb557ca128b6edcb503028c))
* **files:** add WebDAV-oriented Files operations ([ebbd5d5](https://github.com/eSlider/go-onlyoffice/commit/ebbd5d5373abfeca5f160312f201cbd683f55e42))
* **mail:** oo mails send — SendMail + guard empty-by-id send ([dda2bd3](https://github.com/eSlider/go-onlyoffice/commit/dda2bd3ca5cea6ebb89e1ac02b785b9838727bda))
* **mail:** oo mails send — SendMail client + guard empty-by-id ([ab7dfbd](https://github.com/eSlider/go-onlyoffice/commit/ab7dfbd8594de5724286e2b460e789f4acb114bc))
* **security:** secret-scan via gitleaks in CI + pre-push/pre-commit hooks ([#142](https://github.com/eSlider/go-onlyoffice/issues/142)) ([9ead554](https://github.com/eSlider/go-onlyoffice/commit/9ead554f5cc229687d3a27934a023fb1dddaef15))


### Bug Fixes

* **ci:** point gitleaks at /github/workspace in docker action ([2c0df0d](https://github.com/eSlider/go-onlyoffice/commit/2c0df0d55fb03555b60481695554d75dceffce48))
* **files:** rewrite viewUrl host to API base on download ([ecf34ba](https://github.com/eSlider/go-onlyoffice/commit/ecf34ba51aae77f1626a60795f7329f25d9344e5))


### Documentation

* **readme:** document oo docs md↔docx / OCR agent workflow ([239c5ea](https://github.com/eSlider/go-onlyoffice/commit/239c5ea67201e62de6ba067f0e7963c7c9bda188))

## [0.11.0](https://github.com/eSlider/go-onlyoffice/compare/v0.10.0...v0.11.0) (2026-08-18)


### Features

* **catalog:** port oo catalog (scan/merge/match/apply) from legacy branch ([256864f](https://github.com/eSlider/go-onlyoffice/commit/256864f613f31570cf995dddc51255a9a91a38f8))

## [0.10.0](https://github.com/eSlider/go-onlyoffice/compare/v0.9.0...v0.10.0) (2026-08-13)


### Features

* **crm:** add invoices CLI and opportunity update ([19a7e9c](https://github.com/eSlider/go-onlyoffice/commit/19a7e9cbf8f7bd335c4e4cfd4d3b8a84a350cc41))
* **crm:** allow updating invoice terms via oo ([31c556d](https://github.com/eSlider/go-onlyoffice/commit/31c556d34cedd65d6957441f448520af9a7e380b))
* **crm:** invoice update notes and PO fields ([d81e7de](https://github.com/eSlider/go-onlyoffice/commit/d81e7de9114a29439142efba67f205653f562730))
* **crm:** invoices CLI and opportunity update ([f958309](https://github.com/eSlider/go-onlyoffice/commit/f95830961d3aecba65f3cb074d33c9c98f6c6cf2))
* **mail:** draft, attach, and draft-invoice CLI ([73b1050](https://github.com/eSlider/go-onlyoffice/commit/73b1050a83e9e49dcf99d8a966a84746376d7dd9))


### Bug Fixes

* **crm:** JSON person update and oo persons update CLI ([42c31d4](https://github.com/eSlider/go-onlyoffice/commit/42c31d4b485bfab6903aeed0dd6fb2c43f61b4f2))
* **crm:** link invoices to opportunities ([3201a58](https://github.com/eSlider/go-onlyoffice/commit/3201a58df75bceb7d41034970cf384a9169163ed))
* **mail:** German spacing for invoice draft template ([7f35b0e](https://github.com/eSlider/go-onlyoffice/commit/7f35b0e91075d871f87de95b6c84b2353e02f9cf))


### Documentation

* **crm:** capture association graph and invoice/mail quirks ([490426c](https://github.com/eSlider/go-onlyoffice/commit/490426c2779160f1a9b168f6ccae9dd8dccf89e9))
* **crm:** clarify mail API send vs signature for chat links ([6b5a82b](https://github.com/eSlider/go-onlyoffice/commit/6b5a82bef831341aee16b0b0ea1fe6db2c24567d))
* **crm:** Team vs Contacts and persons update note ([11aa23f](https://github.com/eSlider/go-onlyoffice/commit/11aa23f1a17471bad2bd95fddb769a94afbcf97a))

## [0.9.0](https://github.com/eSlider/go-onlyoffice/compare/v0.8.3...v0.9.0) (2026-07-26)


### Features

* **catalog:** mbox header scan, company legal-suffix match, apply order ([8b29be2](https://github.com/eSlider/go-onlyoffice/commit/8b29be2d9bb71154d473f780ad247c5c440c6021))
* **catalog:** scan Thunderbird address books and Gloda contacts ([502b373](https://github.com/eSlider/go-onlyoffice/commit/502b373e3726ded8b9c50e061ff30aea83205a34))
* **oo:** catalog scan/match/apply for CRM clients and contacts ([547f063](https://github.com/eSlider/go-onlyoffice/commit/547f063f7fe922fe784358093a89730f9aaf0683))
* **oo:** catalog scan/match/apply for CRM clients and contacts ([c32e4dd](https://github.com/eSlider/go-onlyoffice/commit/c32e4dd6116f0b8df67e93c5b589a417dfee59aa))
* **projects:** CRM contacts + git author linking ([457c589](https://github.com/eSlider/go-onlyoffice/commit/457c589abdc2c9fe8edd42b4b7445581976cc5a2))
* **projects:** link CRM companies and git authors to projects ([5a8735e](https://github.com/eSlider/go-onlyoffice/commit/5a8735e2898cf53e330e183250cf3c5d3c84da55))


### Bug Fixes

* **catalog:** oo_projects field + safer match ([8d5ca67](https://github.com/eSlider/go-onlyoffice/commit/8d5ca67d64da83c70d8ae293f4c7c23f9f9f59d3))
* **catalog:** preserve oo_id on match; add oo_projects field ([d873cb7](https://github.com/eSlider/go-onlyoffice/commit/d873cb748adf4458369d8a2167abd8da987737c5))
* **crm:** clean person names; canonical project titles ([b9591f5](https://github.com/eSlider/go-onlyoffice/commit/b9591f5fc3efc13b78dbe932ff32c2f2905e18fc))
* **crm:** person names + CC | Company | Title projects ([6305135](https://github.com/eSlider/go-onlyoffice/commit/6305135bd799efe42dbb900a694e82aeae9f9e58))

## [0.8.3](https://github.com/eSlider/go-onlyoffice/compare/v0.8.2...v0.8.3) (2026-07-23)


### Bug Fixes

* **ci:** dispatch GoReleaser after release-please; allow Tests dispatch ([#13](https://github.com/eSlider/go-onlyoffice/issues/13)) ([bf2fa63](https://github.com/eSlider/go-onlyoffice/commit/bf2fa637aaf0419ba5af40b25b64f9ebe24116e8))

## [0.8.2](https://github.com/eSlider/go-onlyoffice/compare/v0.8.1...v0.8.2) (2026-07-23)


### Bug Fixes

* **ci:** retry tag fetch in Release workflow ([#11](https://github.com/eSlider/go-onlyoffice/issues/11)) ([31b4aa0](https://github.com/eSlider/go-onlyoffice/commit/31b4aa0d90393e6d93513284e453b7da7e92c62d))

## [0.8.1](https://github.com/eSlider/go-onlyoffice/compare/v0.8.0...v0.8.1) (2026-07-23)


### Bug Fixes

* **oo:** assign owner and deadline on task create ([#9](https://github.com/eSlider/go-onlyoffice/issues/9)) ([f608207](https://github.com/eSlider/go-onlyoffice/commit/f60820734924284d6c40b46c79380a24573432f3))

## [0.8.0](https://github.com/eSlider/go-onlyoffice/compare/v0.7.0...v0.8.0) (2026-07-23)


### Features

* **office:** users admin UI, table rendering, and user save fixes ([40b74eb](https://github.com/eSlider/go-onlyoffice/commit/40b74eb8abf4d63793a2cb9e59bd899734368b7e))
* **search:** add example SearXNG JSON client ([79b1a6b](https://github.com/eSlider/go-onlyoffice/commit/79b1a6b2c397c2a8f8aeef7e2d8e6d81e0a93db2))
* **search:** add example SearXNG JSON client ([bf144f2](https://github.com/eSlider/go-onlyoffice/commit/bf144f2bb37c44aa37ff33c6b03f59d7b5023dc7))


### Bug Fixes

* **oo:** skip junk dirs in applications Discover ([470ac93](https://github.com/eSlider/go-onlyoffice/commit/470ac938b1e388a17aabc5146cd62070dbb00587))
* **oo:** skip junk dirs in applications Discover ([830f0c7](https://github.com/eSlider/go-onlyoffice/commit/830f0c7208f94521a59dfe6f96d496509e6b29e9))


### Code Refactoring

* **office:** generalize DataTable layout and document TUI table skill ([38932b0](https://github.com/eSlider/go-onlyoffice/commit/38932b0e0eb9696c41a48ad74c943034ebed59e5))

## [0.7.0](https://github.com/eSlider/go-onlyoffice/compare/v0.6.0...v0.7.0) (2026-07-23)


### Features

* **office:** mail preview, infinite scroll, pane resize, and scrollbars ([72844df](https://github.com/eSlider/go-onlyoffice/commit/72844dfe243e04c61ea9bc4db66c55abc08d3ae8))
* **office:** users admin UI, table rendering, and user save fixes ([40b74eb](https://github.com/eSlider/go-onlyoffice/commit/40b74eb8abf4d63793a2cb9e59bd899734368b7e))


### Bug Fixes

* **oo:** skip junk dirs in applications Discover ([470ac93](https://github.com/eSlider/go-onlyoffice/commit/470ac938b1e388a17aabc5146cd62070dbb00587))
* **oo:** skip junk dirs in applications Discover ([830f0c7](https://github.com/eSlider/go-onlyoffice/commit/830f0c7208f94521a59dfe6f96d496509e6b29e9))


### Code Refactoring

* **office:** generalize DataTable layout and document TUI table skill ([38932b0](https://github.com/eSlider/go-onlyoffice/commit/38932b0e0eb9696c41a48ad74c943034ebed59e5))

## [0.6.0](https://github.com/eSlider/go-onlyoffice/compare/v0.5.0...v0.6.0) (2026-06-24)


### Features

* **office:** table detail panes and Alt+1/2/3 layout toggles ([934af21](https://github.com/eSlider/go-onlyoffice/commit/934af21bc93a54df9f40aacb4d89468b00d7562e))


### Documentation

* CHANGELOG for office v0.5.1 release ([2985070](https://github.com/eSlider/go-onlyoffice/commit/29850701f8c9ab0c6d6427f63eaa189a1c558140))

## [0.5.0](https://github.com/eSlider/go-onlyoffice/compare/v0.4.0...v0.5.0) (2026-06-24)


### Features

* **office:** scrollable panes, nav tree drill-down, and item actions ([fe2ee48](https://github.com/eSlider/go-onlyoffice/commit/fe2ee481b9c0f959b798fd40679084a8c4bc4ba2))

## [0.4.0](https://github.com/eSlider/go-onlyoffice/compare/v0.3.2...v0.4.0) (2026-06-24)


### ⚠ BREAKING CHANGES

* **cli:** subject-based command tree (tea-style) + global --output flag
* rename oo-cli → oo, split library by domain, relocate applications

### Features

* **cli:** subject-based command tree (tea-style) + global --output flag ([1e6d22f](https://github.com/eSlider/go-onlyoffice/commit/1e6d22f38dd7a70d4cf1b69ef0b8749b01fded1a))
* **crm:** dedupe duplicates, fix deal titles, and add cleanup CLI ([c77519f](https://github.com/eSlider/go-onlyoffice/commit/c77519fac5ae9dffaad3f1f0ae722d969e62dde9))
* **crm:** merge company slogan variants in dedupe grouping ([3eb649c](https://github.com/eSlider/go-onlyoffice/commit/3eb649c58bf489b221f90734e04f33063f736914))
* **files:** project/task Documents API + oo projects|tasks files ([e03fd62](https://github.com/eSlider/go-onlyoffice/commit/e03fd6220012c73b110f1e0b02a12971174eb290))
* **mails:** add Workspace mail CLI with pagination and parsed from fields ([1cb228e](https://github.com/eSlider/go-onlyoffice/commit/1cb228e5d83ab453660bbace5f936a77f9381fc2))
* **office:** add Workspace TUI with shared bootstrap and test suite ([7358ff5](https://github.com/eSlider/go-onlyoffice/commit/7358ff5c326cdc873cb68b3f96d31155ecba1ee3))


### Code Refactoring

* rename oo-cli → oo, split library by domain, relocate applications ([cff8145](https://github.com/eSlider/go-onlyoffice/commit/cff8145f8c35656a77b7a36338428bf9be0eda0f))


### Documentation

* CHANGELOG 0.6.0, README, AGENTS.md, cmd/oo/main.go tree. ([e03fd62](https://github.com/eSlider/go-onlyoffice/commit/e03fd6220012c73b110f1e0b02a12971174eb290))

## [Unreleased]

## [0.7.0] — 2026-06-24

### Added — `office` TUI

- **Pane layout** — default **10% / 60% / 30%** split (nav / list / detail); drag vertical borders to resize; **Alt+1/2/3** still toggles panes.
- **Filter mode** (`f`) — live filter on nav + list; right pane is the search query; **Esc** clears.
- **Mail preview** — HTML bodies rendered colorized in the terminal (glamour); read-only scrollable document pane.
- **Mail infinite scroll** — loads the next page automatically when the cursor nears the end of the list.
- **Scrollbars** — vertical scrollbar on nav, list body, and detail preview when content exceeds the viewport.
- **Calendar** — unified calendars + events leaf with **Type** column and date range (−7d … +30d).
- **Tasks** — humanized status labels and relative deadlines in the list.
- **Projects** — open/closed row colors; status toggle in the detail form; **Save** sends `responsibleId` and status via `UpdateProjectStatus`.

### Changed — `office` TUI

- Flattened navigation: **Projects**, **Tasks**, **By project**, **Calendar**, **CRM**, **Mail**, **Users** (removed Browse; Calendar no longer splits Calendars/Events).
- Middle pane truncates overflowing cells with `…`; full text on the cursor or Space-selected row.
- Detail form tab order: Title → Description → Status → Save → Delete.
- **j/k** scrolls mail/file preview and read-only detail forms; mouse wheel scrolls detail when hovering the content area.

### Added — library

- `UpdateProjectStatus` — `PUT /api/2.0/project/{id}/status` for open/closed lifecycle.

## [0.5.1] — 2026-06-24

### Added — `office` TUI

- **Alt+1 / Alt+2 / Alt+3** — show/hide left (nav), middle (table), and right (detail) panes.
- Visible panes share **100% terminal width** evenly; pane content fills its column.
- Tab focus skips hidden panes.

### Changed — `office` TUI

- Split detail pane: form/document top (~72%), CRUD action bar bottom.
- Middle pane: multi-column table with sort, selection, and full-width columns.
- Project list columns: ID, Title, Tasks (open/closed), Documents, Users.
- Row selection auto-loads detail; files show document preview, entities show forms.

## [0.6.0] - 2026-04-24

### Added — library

- **Project & task Documents API** in [`files.go`](files.go):
  - `FileEntry`, `FolderEntry`, `ProjectFilesResponse` types.
  - `GetProjectFiles`, `GetTaskFiles`, `GetFile`.
  - `UploadProjectFile` — `POST /api/2.0/files/{folderId}/upload` into the
    project's `projectFolder` (resolved via `GetProjectByID` or first folder
    from `GetProjectFiles`).
  - `AttachFilesToTask` — `POST .../project/task/{id}/files` with form
    `files=<id>` (OnlyOffice expects **existing** file ids, not multipart).
  - `UploadTaskFile` — uploads via `UploadProjectFile` using the task's
    `projectOwner.id`, then attaches.
  - `DetachTaskFile` — `DELETE .../files?fileid=`.
  - `RenameFile` — `PUT /api/2.0/files/file/{id}.json` with JSON body.
  - `DeleteFiles` — `PUT /api/2.0/files/fileops/delete.json` with `fileIds`.
  - `DownloadFile` — `GetFile` then `GET` on `viewUrl` with `Authorization`.
  - Helpers: `FileEntryNumericID`, `FileEntryTitle`, `SafeLocalFileName`.
- **`putJSON`** on `*Client` in [`http.go`](http.go) for JSON PUT bodies.

### Added — CLI

- `oo projects files list|upload|download|rename|delete` — see
  [`cmd/oo/projects_files.go`](cmd/oo/projects_files.go); `list` supports
  `--folders`.
- `oo tasks files list|upload|detach` — see [`cmd/oo/tasks_files.go`](cmd/oo/tasks_files.go).

### Added — tests

- [`files_integration_test.go`](files_integration_test.go) — live roundtrip
  against OnlyOffice (same credential rules as `client_test.go`).
- [`files_test.go`](files_test.go) + `testdata/*.json` — envelope decode unit
  tests (no network).

## [0.5.0] - 2026-04-24

### Changed — CLI **BREAKING**

- **Subject-based command tree** (`tea`-style). The flat `oo verb-noun`
  layout is replaced with `oo <subject> <verb>`:

  | Old | New |
  |---|---|
  | `oo cal-list` | `oo calendar list` |
  | `oo cal-events` | `oo calendar events` |
  | `oo cal-add` / `cal-delete` | `oo calendar add` / `calendar delete` |
  | `oo task-list` | `oo tasks list` |
  | `oo task-add` | `oo tasks create` |
  | `oo task-update` | `oo tasks update` (deletion moved to `oo tasks delete`) |
  | `oo subtask-add` | `oo tasks subtask add` |
  | `oo crm-contacts` | `oo contacts list` (plus filtered `oo persons list` / `oo companies list`) |
  | `oo crm-add-contact --company …` | `oo companies create --name …` |
  | `oo crm-add-contact --person-first …` | `oo persons create --first …` |
  | `oo crm-deals` | `oo opportunities list` |
  | `oo crm-deals --stages` | `oo opportunities stages` |
  | `oo crm-add-deal` | `oo opportunities create` |
  | `oo crm-cases` | `oo cases list` |
  | `oo applications-sync` | `oo applications sync` |

- **New subjects**: `oo projects {list,get,milestones,create,update,delete}`,
  `oo users {list,self}` (plus top-level `oo whoami`), `oo crm-tasks
  {list,create,delete,categories}`, `oo cases {create,delete,member-add}`,
  `oo contacts {get,info-add}`.

- **Global `--output/-o` flag**: all list-like commands now support
  `--output table` (default; tabwriter-aligned, truncated to 80 chars
  per cell) and `--output json`. Nested `bidCurrency` flattened to its
  `abbreviation` in the table view.

- **Module aliases**: `oo calendar|cal`, `oo projects|prj`, `oo tasks|task`,
  `oo persons|person`, `oo companies|company`, `oo opportunities|deals|deal`,
  `oo cases|case`, `oo applications|apps`. `delete|rm` on every leaf that
  removes things.

### Added

- `cmd/oo/common.go` — shared `printTable(headers, rows)` and
  `printObject(v)` helpers that dispatch on the `--output` flag.
- `cmd/oo/users.go` — exposes `oo users list`, `oo users self`, `oo whoami`
  via the library's `GetUsers` + `SelfUserID`.
- `cmd/oo/projects.go` — full CRUD for projects backed by `CreateProject`,
  `UpdateProject`, `DeleteProject`, `GetProjectByID`, `GetProjectMilestones`.
- `cmd/oo/crm_tasks.go` — dedicated `oo crm-tasks` subject (distinct from
  project `oo tasks`).
- `cmd/oo/contacts.go` — unified contacts/persons/companies with shared
  list/filter implementation.

### Changed — library (minor)

- `newOO` in `cmd/oo/common.go` now calls `AuthenticateContext(cmd.Context())`
  so CLI aborts propagate to the auth request.

## [0.4.0] - 2026-04-24

### Changed — project structure

- **Library files reorganised by domain** (mechanical split; zero API surface
  change). The former monolithic `onlyoffice.go` (687 LOC) is now split into:
  - `client.go` — `Client`, `Credentials`, `Defaults`, env helpers, `NewClient`.
  - `request.go` — `Request`, `Query`, `Time`, `Token`, `MetaResponse`,
    `Permissions`, `requestBodyReader`.
  - `auth.go` — `Authenticate`, `AuthenticateContext`, `InvalidateToken`,
    `Auth`, `ensureToken`, `authHeader`, `tokenValid`.
  - `http.go` — transport helpers and DRY response decoders
    (`ResponseArray`, `ResponseObject`, `postFormObject`, `putFormObject`,
    `deleteObject`, `unmarshalResponseObject`).
  - `projects.go` — `Project`, `Projects`, `Milestone`, `ProjectOwner` +
    `GetProjects` / `CreateProject` / `UpdateProject` / `DeleteProject` /
    `GetProjectByID` / `GetProjectMilestones`.
  - `tasks.go` — `Task`, `ProjectTaskStatus`, `TaskPriority`,
    `ProjectGetTasksRequest` et al. **plus** the form-encoded helpers
    formerly in `tasks_extra.go` (`ListTasks`, `AddTask`, `AddSubtask`,
    `UpdateTaskStatus`, `DeleteTask`, `GetTaskByID`).
  - `users.go` — `User`, `Contact`, `Group`, `GetUsers`, `SelfUserID`.
  - `calendar.go`, `crm.go`, `files.go` — unchanged in scope, refactored
    through the new DRY helpers.
  - `httpx.go` → **renamed** `http.go`.
  - `onlyoffice.go` and `tasks_extra.go` — **deleted** (content redistributed).

### Changed — CLI **BREAKING**

- **Binary renamed `oo-cli` → `oo`.** Install with
  `go install github.com/eslider/go-onlyoffice/cmd/oo@latest`.
- **Package path `cmd/oo-cli` → `cmd/oo`.** The old path is removed.
- **`internal/cli` is gone.** Cobra commands now live directly under
  `cmd/oo/` as `package main`, split by domain: `calendar.go`, `crm.go`,
  `tasks.go`, `apps.go`, `common.go`. Rationale: cobra wiring is a CLI-only
  concern and does not belong inside a `pkg-level internal/`.
- **`internal/applications` → `cmd/oo/applications/`.** This is a
  CV-specific CRM workflow — not a general OnlyOffice feature — and is only
  consumed by the `oo` CLI. Keeping it under `cmd/oo/` prevents accidental
  external adoption and makes the coupling explicit.
- **`examples/applications/` removed.** It imported an internal package,
  which was a policy smell. Remaining examples (`basic`, `calendar`, `crm`,
  `subtasks`) use only the exported library surface.

### Added

- `(*Client).ResponseObject` — GET-and-decode-object counterpart to the
  existing `ResponseArray`.
- `(*Client).postFormObject` / `putFormObject` / `deleteObject` — eliminate
  the ~15 identical "form request → `responseField` → `json.Unmarshal`"
  blocks previously duplicated across `crm.go` / `tasks_extra.go` /
  `calendar.go` / `files.go`.

### Migration

External library consumers: **no changes required**. The module path
(`github.com/eslider/go-onlyoffice`), the `onlyoffice` package name, and
every exported symbol are unchanged.

CLI users: replace `oo-cli` with `oo` in scripts and CI. The command set
and flags are identical.

## [0.3.2] - 2026-04-24

### Fixed

- `Project.String()` no longer interprets the title as a format string
  (`fmt.Sprintf(*p.Title)`) and is now nil-safe on a zero-value `Project`.
- `internal/applications.buildSummary` no longer panics at regex compile time
  on Go 1.23+ — the previous `(?= ...)` lookahead is replaced with an RE2-safe
  non-capturing trailing delimiter.

### Changed

- `Client.Query` now routes token acquisition through the shared
  `ensureToken` path instead of duplicating the auth-expiry check inline.
- Request body marshalling is consolidated into an unexported
  `requestBodyReader` helper (DRY; no change to the public surface).
- The `Request.Debug` field is preserved for backwards compatibility but no
  longer changes behaviour — both branches used to unmarshal into the same
  target value. We'll remove the field in a future major release.

### Tests

- Deleted `httptest.NewServer` fixtures that emulated OnlyOffice protocol
  endpoints. Replaced them with:
  - pure-Go unit tests in `unit_test.go` (no network);
  - real integration tests in `client_test.go` guarded by
    `//go:build integration`. Run with
    `go test -tags=integration ./...`. Tests skip cleanly when
    `ONLYOFFICE_URL/USER/PASS` (or aliases) are absent.
- New policy documented in `AGENTS.md` and
  `.cursor/rules/no-synthetic-mocks.mdc`.

## [0.3.1] - 2026-04-24

### Added

- `AuthenticateContext(ctx)` — context-aware auth that honours cancellation and
  deadlines. Preferred entry point for long-running syncs (cron, watchers).
- `InvalidateToken()` — clears the cached token to force re-auth on the next
  request. Use this to recover from a mid-sync 401 when the server has revoked
  the session while the local `Expires` timestamp still looks fresh.

### Notes

- Plain `Authenticate()` is unchanged and remains a convenience wrapper around
  `AuthenticateContext(context.Background())`.
- No breaking changes; a patch release.

## [0.3.0] - 2026-04-24

### Added

- Calendar helpers: `ListCalendars`, `ListEvents`, `AddEvent`, `DeleteEvent`.
- CRM helpers: contacts (`ListContacts`, `GetContact`, `FindCompany`, `FindPerson`,
  `CreateCompany`, `CreatePerson`, `AddContactInfo`, `DeleteContact`), deals
  (`ListOpportunities`, `GetOpportunity`, `CreateOpportunity`,
  `AddOpportunityMember`, `ListDealStages`, `DeleteOpportunity`), cases
  (`ListCases`, `CreateCase`, `AddCaseMember`, `DeleteCase`), CRM tasks
  (`ListCRMTasks`, `CreateCRMTask`, `DeleteCRMTask`, `ListTaskCategories`), and
  history notes (`AddHistoryNote`).
- Project task extras: `GetProjectByID`, `ListTasks`, `ListAllTasks`,
  `GetTaskByID`, `AddTask`, `AddSubtask`, `UpdateTaskStatus`, `DeleteTask`.
- File upload: `UploadOpportunityFile` (multipart).
- `SelfUserID` cached lookup of `people/@self`.
- `Defaults` struct + `SetDefaults` + `GetEnvironmentDefaults` for optional
  calendar/project fallbacks.
- Alias env vars accepted by `GetEnvironmentCredentials`:
  `ONLYOFFICE_HOST` / `ONLYOFFICE_NAME` / `ONLYOFFICE_PASSWORD`.
- Public `Authenticate()` that primes the token eagerly.
- Bundled CLI: `cmd/oo-cli` (Cobra) with commands `cal-list`, `cal-events`,
  `cal-add`, `cal-delete`, `task-list`, `task-add`, `subtask-add`,
  `task-update`, `crm-contacts`, `crm-add-contact`, `crm-deals`,
  `crm-add-deal`, `crm-cases`, `applications-sync`.
- httptest-based unit tests for form / multipart / CRM helpers.

### Changed

- The `onlyoffice.Client` struct gained unexported fields (`defaults`, `selfID`,
  `noteCatID`); the public API is unchanged and remains backwards compatible.

## [0.2.0] - earlier

- Badges, docs expansion (Gitea sync use case, Gantt/PM workflows).

## [0.1.0] - earlier

- Initial release: OnlyOffice Project Management API client.
