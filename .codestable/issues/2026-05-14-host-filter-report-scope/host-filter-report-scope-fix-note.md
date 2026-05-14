---
doc_type: issue-fix
issue: 2026-05-14-host-filter-report-scope
status: completed
severity: P1
tags: [host-filter, report, n9e, victoriametrics, excel]
created_at: "2026-05-14"
related:
  - host-filter-report-scope-report.md
  - host-filter-report-scope-analysis.md
---

# Host Filter 报告范围修复记录

## 修复摘要

修复 `inspection.host_filter` 只过滤 VM 指标、不收敛 N9E 主机元信息导致报告包含夜莺全量机器的问题。修复后，Host 报告主机列表按 VM 过滤后的实际指标范围输出。

验证运行中还发现 Redis 多集群 Excel sheet 名可能超过 31 字符，导致 Excel 报告生成失败；该阻断点已一并修复。

## 代码改动

- `internal/service/collector.go`
  - 在 `CollectAll` 中对启用 `host_filter` 的 Host 巡检按 VM 实际非 N/A 指标结果收敛 N9E 主机列表。
  - 当 business group + tags 查询无指标、但 tags 存在时，记录 warning 并使用 tags-only fallback，适配当前 VM 指标无 `busigroup` 标签但有 `items` 标签的环境。
- `internal/service/collector_test.go`
  - 增加 HostFilter 报告范围收敛回归测试。
  - 增加 business group 无命中时 tag-only fallback 回归测试。
- `internal/report/excel/writer.go`
  - Redis 多集群 sheet 名规范化、截断到 31 字符并处理重名。
- `internal/report/excel/writer_test.go`
  - 增加长 Redis cluster ID 的 Excel sheet 名回归测试。

## 验证

- `env -u GOROOT go test ./internal/service -count=1`：通过。
- `env -u GOROOT go test ./internal/report/excel -count=1`：通过。
- `env -u GOROOT go test ./...`：通过。
- `env -u GOROOT make build`：通过，生成 `bin/inspect`。
- `./bin/inspect run -c config.yaml`：已复跑。
  - N9E target：764。
  - 修复后 Host 报告主机数：52。
  - Host failed：0。
  - Excel 报告：`reports/inspection_report_2026-05-14.xlsx` 生成成功。
  - HTML 报告：`reports/inspection_report_2026-05-14.html` 生成成功。

## 剩余风险

- 当前命令退出码为 2，是巡检结果中存在 critical 项导致，不是报告生成失败。
- MySQL / Redis / Tomcat 等中间件域有自己的 `instance_filter`，本次只修复 Host 报告范围和 Excel sheet 名阻断点；如需所有中间件也统一继承 Host 的 `items` 范围，需要另开需求或 issue 明确配置语义。
