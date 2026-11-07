#!/usr/bin/env python3
"""
ReqMango 海量数据种子脚本 - 通过 PostgreSQL 直接批量插入
目标：~5000 issues，with comments, activities, time tracks, relations 等完整关系数据
"""

import psycopg2
import psycopg2.extras
import random
import datetime
import hashlib
import sys

def connect():
    return psycopg2.connect(
        host="localhost", port=5432, dbname="reqmango",
        user="postgres", password="postgres"
    )

def weighted_index(weights, rng):
    total = sum(weights)
    r = rng.randint(0, total - 1)
    cum = 0
    for i, w in enumerate(weights):
        cum += w
        if r < cum:
            return i
    return len(weights) - 1

def main():
    conn = connect()
    conn.autocommit = False
    cur = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
    print("=== ReqMango 海量数据种子脚本 ===\n")

    # 1. Fetch existing data
    cur.execute("SELECT id, email, username, display_name, is_superuser FROM users ORDER BY id")
    users = cur.fetchall()
    print(f"[OK] Users: {len(users)}")

    cur.execute("SELECT id, name, slug, owner_id FROM workspaces ORDER BY id")
    workspaces = cur.fetchall()
    print(f"[OK] Workspaces: {len(workspaces)}")

    cur.execute("SELECT id, name, identifier, workspace_id FROM projects ORDER BY id")
    projects = cur.fetchall()
    print(f"[OK] Projects: {len(projects)}")

    # States/cycles/modules/labels per project
    cur.execute("SELECT id, name, color, \"group\", project_id FROM states WHERE is_active=true ORDER BY project_id, sequence")
    all_states = cur.fetchall()
    states_by_project = {}
    for s in all_states:
        states_by_project.setdefault(s["project_id"], []).append(s)

    cur.execute("SELECT id, name, project_id FROM cycles ORDER BY project_id, id")
    all_cycles = cur.fetchall()
    cycles_by_project = {}
    for c in all_cycles:
        cycles_by_project.setdefault(c["project_id"], []).append(c)

    cur.execute("SELECT id, name, project_id FROM modules ORDER BY project_id, id")
    all_modules = cur.fetchall()
    modules_by_project = {}
    for m in all_modules:
        modules_by_project.setdefault(m["project_id"], []).append(m)

    cur.execute("SELECT id, name, color, project_id FROM labels ORDER BY project_id, id")
    all_labels = cur.fetchall()
    labels_by_project = {}
    for l in all_labels:
        labels_by_project.setdefault(l["project_id"], []).append(l)

    cur.execute("SELECT id, workspace_id FROM relation_types ORDER BY id")
    all_rel_types = cur.fetchall()
    rel_types_by_workspace = {}
    for rt in all_rel_types:
        rel_types_by_workspace.setdefault(rt["workspace_id"], []).append(rt)

    # Count existing issues
    cur.execute("SELECT project_id, COUNT(*) as cnt FROM issues GROUP BY project_id ORDER BY project_id")
    existing = {r["project_id"]: r["cnt"] for r in cur.fetchall()}
    for p in projects:
        print(f"  {p['identifier']:12s} id={p['id']}  existing={existing.get(p['id'], 0)}")

    admin_id = users[0]["id"]

    # ========================================================================
    # TITLE POOLS
    # ========================================================================
    feature_titles = [
        "用户单点登录(SSO)集成", "数据导出报表功能", "批量操作工作项",
        "自定义仪表盘", "邮件通知模板配置", "第三方API集成",
        "工作流可视化编辑器", "高级搜索过滤器", "评论@提及功能",
        "文件在线预览", "甘特图拖拽调整", "时间线视图",
        "项目模板导入导出", "自动化规则引擎", "多维数据透视表",
        "Markdown图表支持", "快捷键自定义", "深色模式切换",
        "看板泳道配置", "跨项目依赖关系图", "实时协作编辑",
        "Webhook事件订阅", "API速率限制配置", "自定义字段公式计算",
        "批量导入CSV/Excel", "日历订阅(iCal)", "附件版本管理",
        "移动端适配优化", "离线模式支持", "屏幕阅读器无障碍",
        "多租户数据隔离", "微前端架构迁移", "GraphQL API层",
        "实时消息推送(WebSocket)", "审计日志查询", "权限细粒度控制",
        "自定义工作流模板", "智能搜索推荐", "数据脱敏展示",
        "全文检索增强", "报表定时邮件", "团队工作日历",
        "外部系统数据同步", "OAuth2.0提供商集成", "SAML单点登录",
        "双因素认证(2FA)", "IP白名单访问控制", "会话管理面板",
        "系统健康检查仪表盘", "数据库备份自动调度", "弹性伸缩策略配置",
        "灰度发布工作流", "A/B测试集成", "功能开关管理",
    ]
    bug_titles = [
        "修复登录页面密码可见性切换失效", "分页组件在大数据量下卡顿",
        "修复文件上传时文件类型校验绕过", "甘特图时间刻度显示错位",
        "通知邮件中链接无法点击", "修复看板拖拽后状态未保存",
        "移动端表格横向滚动异常", "修复日期选择器在Safari中无法使用",
        "API返回500当参数为空数组时", "富文本编辑器图片粘贴失败",
        "修复工作流转换时权限校验缺失", "修复批量删除后列表未刷新",
        "导出Excel中自定义字段缺失", "修复并发编辑冲突未提示",
        "修复搜索时特殊字符导致报错", "日历视图周模式显示错误",
        "修复通知中心未读数不更新", "修复模块删除后关联issue未清理",
        "修复图表在数据为空时崩溃", "修复列表横向滚动条不显示",
        "修复树形视图懒加载死循环", "Dark模式切换主题色丢失",
        "修复Markdown预览XSS注入", "附件删除后存储文件未清理",
        "修复过滤器条件组合逻辑错误", "评论排序在高并发下错乱",
        "修复时区转换夏令时偏差", "修复批量导入时字段映射错误",
        "修复状态转换历史记录丢失", "修复SSE推送连接泄漏",
        "修复内存泄漏导致OOM", "修复千条以上Issue列表加载超时",
        "子任务循环引用未检测", "修复邀请链接过期后仍可注册",
        "修复API文档Swagger路径错误", "修复自定义字段删除后数据残留",
        "修复通知卸载后浏览器报错", "修复页码参数注入SQL风险",
        "修复文件下载文件名编码问题", "修复草稿自动保存覆盖风险",
        "修复Webhook重试次数未限制", "修复标签删除关联清理不完整",
        "修复关系图节点位置漂移", "修复评论@提及正则匹配多字节字符",
        "修复日期范围查询边界条件", "修复移动端手势冲突",
        "修复键盘快捷键在输入框中触发", "修复缓存策略导致脏数据",
        "修复并发创建编号冲突",
    ]
    task_titles = [
        "编写API接口文档", "数据库索引优化", "前端组件单元测试覆盖",
        "代码规范ESLint配置", "CI/CD流水线优化", "依赖包安全升级",
        "日志采集与集中存储", "Redis缓存策略调整", "Nginx反向代理配置",
        "Docker镜像体积优化", "Kubernetes健康检查配置", "数据库慢查询分析",
        "前端打包体积优化", "静态资源CDN部署", "SSL证书自动续期",
        "监控指标Dashboard创建", "告警规则阈值调整", "备份恢复演练",
        "性能基准测试", "安全漏洞修复", "i18n翻译文件整理",
        "数据库主从同步验证", "消息队列消费者实现", "分布式锁接入",
        "链路追踪集成(OpenTelemetry)", "配置文件加密存储", "服务网格配置",
        "前端错误边界处理", "接口幂等性改造", "分布式事务调研",
        "密码策略强化", "Session管理Redis化", "定时任务迁移",
        "前端埋点方案设计", "移动端PWA支持", "DNS域名切换演练",
        "数据库连接池参数调优", "前端懒加载路由拆分", "JWT token刷新机制",
        "前端骨架屏组件开发", "API版本兼容策略", "第三方服务降级方案",
        "存储空间清理脚本", "前端无障碍改进", "代码覆盖率提升至80%",
        "前端E2E测试添加", "操作日志保留策略制定", "灾难恢复预案编写",
        "前端微组件拆分", "API网关限流规则配置", "服务健康检查与自愈",
        "前端首屏性能优化", "数据库分区表方案设计", "数据归档策略实现",
    ]
    story_titles = [
        "作为用户，我希望能通过OAuth登录系统", "作为PM，我希望能批量指派任务",
        "作为开发者，我希望能查看代码关联的issue", "作为测试，我希望能一键提交Bug报告",
        "作为管理员，我希望能自定义用户角色权限", "作为Viewer，我希望能自定义看板视图",
        "作为团队成员，我希望能收到实时通知", "作为项目经理，我希望能查看Sprint进度报告",
        "作为客户，我希望能通过链接查看项目进度", "作为运营，我希望能导出数据到BI工具",
        "作为Reviewer，我希望能批量审批工作项", "作为新成员，我希望能有新手引导",
        "作为团队Lead，我希望能看到代码提交与Issue关联", "作为设计师，我希望能附加Figma文件",
        "作为QA，我希望能关联测试用例到Issue", "作为DevOps，我希望能一键部署关联的Issue",
        "作为产品经理，我希望能拖拽排优先级", "作为CS，我希望能查看客户反馈对应的Issue",
        "作为架构师，我希望能查看系统依赖关系图", "作为财务，我希望能导出工时统计报表",
    ]
    epic_titles = [
        "Q3平台核心能力建设", "国际化与多语言支持", "移动端体验全面升级",
        "AI智能助手集成", "企业级安全合规", "性能优化专项",
        "开发者体验提升", "数据平台建设", "开放生态构建",
        "运维自动化平台", "用户体验重构", "技术架构升级",
        "客户项目交付标准化", "测试自动化覆盖", "监控告警体系完善",
    ]

    priorities = ["urgent", "high", "medium", "low", "none"]
    priority_weights = [5, 15, 35, 35, 10]

    # ========================================================================
    # 2. CREATE ISSUES (target ~500 per project)
    # ========================================================================
    print("\n--- Step 1: Creating Issues ---")
    target_per_project = {p["id"]: 500 for p in projects}
    all_new_entries = {}  # pid -> [(seq, state_id, start_date, target_date, completed_at, prio, title, type_tag, issue_id), ...]
    all_issue_ids = {}    # pid -> [issue_id]

    for proj in projects:
        pid = proj["id"]
        ws_id = proj["workspace_id"]
        states = states_by_project.get(pid, [])
        cycles = cycles_by_project.get(pid, [])
        modules = modules_by_project.get(pid, [])
        labels = labels_by_project.get(pid, [])

        if not states or not modules or not labels:
            print(f"  SKIP {proj['identifier']}: missing states/modules/labels")
            continue

        existing_cnt = existing.get(pid, 0)
        to_create = max(0, target_per_project[pid] - existing_cnt)
        if to_create == 0:
            print(f"  SKIP {proj['identifier']}: already has {existing_cnt} issues (>= 500)")
            continue

        cur.execute("SELECT COALESCE(MAX(sequence_id), 0) as max_seq FROM issues WHERE project_id = %s", (pid,))
        max_seq = cur.fetchone()["max_seq"]

        print(f"  {proj['identifier']:12s} id={pid}  {existing_cnt}→{target_per_project[pid]}  (+{to_create})  seq_start={max_seq+1}")

        rng = random.Random(pid * 997 + max_seq)
        entries = []
        batch = []
        batch_size = 100

        for i in range(to_create):
            seq = max_seq + i + 1
            roll = rng.randint(0, 99)

            if roll < 25:
                title = feature_titles[rng.randint(0, len(feature_titles)-1)]
                type_tag = "Feature"
            elif roll < 45:
                title = bug_titles[rng.randint(0, len(bug_titles)-1)]
                type_tag = "Bug"
            elif roll < 65:
                title = task_titles[rng.randint(0, len(task_titles)-1)]
                type_tag = "Task"
            elif roll < 80:
                title = story_titles[rng.randint(0, len(story_titles)-1)]
                type_tag = "Story"
            elif roll < 92:
                title = f"Epic: {epic_titles[rng.randint(0, len(epic_titles)-1)]}"
                type_tag = "Epic"
            else:
                title = f"技术调研: {feature_titles[rng.randint(0, len(feature_titles)-1)]}"
                type_tag = "Spike"

            h = hashlib.md5(f"{pid}-{seq}-{rng.random()}".encode()).hexdigest()[:4]
            title = f"{title} [{h}]"

            sid = weighted_index([15, 20, 25, 15, 20, 5], rng)
            selected_state = states[sid] if sid < len(states) else states[-1]

            prio = priorities[weighted_index(priority_weights, rng)]

            days_ago = rng.randint(1, 180)
            start_date = datetime.date.today() - datetime.timedelta(days=days_ago)
            target_date = None
            completed_at = None
            if rng.randint(0, 99) < 60:
                td = start_date + datetime.timedelta(days=rng.randint(3, 30))
                target_date = td.isoformat()
            if selected_state["group"] in ("completed", "cancelled"):
                ct = start_date + datetime.timedelta(days=rng.randint(1, 14))
                completed_at = ct.isoformat()

            desc_html = f"<h2>概述</h2><p>{proj['name']} 中的 {type_tag} 类型工作项。</p><h3>验收标准</h3><ul><li>功能满足需求规格</li><li>单元测试覆盖通过</li><li>代码审查通过</li></ul><h3>技术备注</h3><p>创建于 {start_date}</p>"
            desc_stripped = f"{proj['name']} {type_tag} - 验收标准: 功能满足规格, 测试通过, 审查通过"
            sort_order = round(rng.random() * 65536, 2)

            batch.append((
                title, desc_html, desc_stripped, prio, seq,
                selected_state["id"], pid, ws_id,
                start_date.isoformat(), target_date, completed_at,
                sort_order, admin_id
            ))

            # Save entry info for later (without issue_id yet)
            entries.append((seq, selected_state["id"], start_date.isoformat(),
                            target_date, completed_at, prio, title, desc_html,
                            desc_stripped, type_tag))

            if len(batch) >= batch_size or i == to_create - 1:
                psycopg2.extras.execute_values(cur, """
                    INSERT INTO issues (name, description_html, description_stripped, priority, sequence_id,
                        state_id, project_id, workspace_id, start_date, target_date, completed_at,
                        sort_order, created_by_id, created_at, updated_at)
                    VALUES %s
                    ON CONFLICT DO NOTHING
                    RETURNING id
                """, batch, template="(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, NOW(), NOW())")

                returned = cur.fetchall()
                # Match returned IDs to entries (some may have been skipped by ON CONFLICT)
                offset = len(entries) - len(batch)
                for j, row in enumerate(returned):
                    entries[offset + j] = entries[offset + j] + (row["id"],)

                conn.commit()
                batch = []

        # Only keep entries that got an issue_id
        valid_entries = [e for e in entries if len(e) > 10]
        ids = [e[-1] for e in valid_entries]
        all_new_entries[pid] = valid_entries
        all_issue_ids[pid] = ids
        print(f"    Created {len(ids)} issues")

    total_issues_created = sum(len(v) for v in all_issue_ids.values())
    print(f"\nTotal new issues: {total_issues_created}")

    # ========================================================================
    # 3. ASSIGNEES (1-3 per issue) -- join table, no timestamps
    # ========================================================================
    print("\n--- Step 2: Creating Assignees ---")
    total_asgn = 0
    batch = []
    for pid, ids in all_issue_ids.items():
        rng = random.Random(pid * 311 + 19)
        for iid in ids:
            n = 1 + rng.randint(0, 2)
            shuffled = list(users)
            rng.shuffle(shuffled)
            for j in range(min(n, len(shuffled))):
                batch.append((iid, shuffled[j]["id"]))
                total_asgn += 1
                if len(batch) >= 500:
                    psycopg2.extras.execute_values(cur, """
                        INSERT INTO issue_assignees (issue_id, user_id)
                        VALUES %s
                        ON CONFLICT DO NOTHING
                    """, batch, template="(%s, %s)")
                    conn.commit()
                    batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO issue_assignees (issue_id, user_id)
            VALUES %s
            ON CONFLICT DO NOTHING
        """, batch, template="(%s, %s)")
        conn.commit()
    print(f"  Created {total_asgn} assignee links")

    # ========================================================================
    # 4. LABELS (1-4 per issue) -- join table, no timestamps
    # ========================================================================
    print("\n--- Step 3: Creating Labels ---")
    total_il = 0
    batch = []
    for pid, ids in all_issue_ids.items():
        labels = labels_by_project.get(pid, [])
        if not labels:
            continue
        rng = random.Random(pid * 411 + 23)
        for iid in ids:
            n = 1 + rng.randint(0, 3)
            shuffled = list(labels)
            rng.shuffle(shuffled)
            for j in range(min(n, len(shuffled))):
                batch.append((iid, shuffled[j]["id"]))
                total_il += 1
                if len(batch) >= 500:
                    psycopg2.extras.execute_values(cur, """
                        INSERT INTO issue_labels (issue_id, label_id)
                        VALUES %s
                        ON CONFLICT DO NOTHING
                    """, batch, template="(%s, %s)")
                    conn.commit()
                    batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO issue_labels (issue_id, label_id)
            VALUES %s
            ON CONFLICT DO NOTHING
        """, batch, template="(%s, %s)")
        conn.commit()
    print(f"  Created {total_il} label links")

    # ========================================================================
    # 5. CYCLES (60% of issues) -- join table, no timestamps
    # ========================================================================
    print("\n--- Step 4: Creating Issue Cycles ---")
    total_ic = 0
    batch = []
    for pid, ids in all_issue_ids.items():
        cycles = cycles_by_project.get(pid, [])
        if not cycles:
            continue
        rng = random.Random(pid * 511 + 29)
        for iid in ids:
            if rng.randint(0, 99) < 60:
                c = cycles[rng.randint(0, len(cycles)-1)]
                batch.append((iid, c["id"]))
                total_ic += 1
                if len(batch) >= 500:
                    psycopg2.extras.execute_values(cur, """
                        INSERT INTO issue_cycles (issue_id, cycle_id)
                        VALUES %s
                        ON CONFLICT DO NOTHING
                    """, batch, template="(%s, %s)")
                    conn.commit()
                    batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO issue_cycles (issue_id, cycle_id)
            VALUES %s
            ON CONFLICT DO NOTHING
        """, batch, template="(%s, %s)")
        conn.commit()
    print(f"  Created {total_ic} cycle links")

    # ========================================================================
    # 6. MODULES (80% of issues) -- join table, no timestamps
    # ========================================================================
    print("\n--- Step 5: Creating Module Issues ---")
    total_mi = 0
    batch = []
    for pid, ids in all_issue_ids.items():
        modules = modules_by_project.get(pid, [])
        if not modules:
            continue
        rng = random.Random(pid * 611 + 31)
        for iid in ids:
            if rng.randint(0, 99) < 80:
                m = modules[rng.randint(0, len(modules)-1)]
                batch.append((m["id"], iid))
                total_mi += 1
                if len(batch) >= 500:
                    psycopg2.extras.execute_values(cur, """
                        INSERT INTO module_issues (module_id, issue_id)
                        VALUES %s
                        ON CONFLICT DO NOTHING
                    """, batch, template="(%s, %s)")
                    conn.commit()
                    batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO module_issues (module_id, issue_id)
            VALUES %s
            ON CONFLICT DO NOTHING
        """, batch, template="(%s, %s)")
        conn.commit()
    print(f"  Created {total_mi} module links")

    # ========================================================================
    # 7. COMMENTS (~50% of issues, 1-5 each) -- has BaseModel
    # ========================================================================
    print("\n--- Step 6: Creating Comments ---")
    comment_texts = [
        "这个我来处理，优先级可以调高一些。",
        "已经和相关团队确认过了，可以按这个方案推进。",
        "需要在周三之前完成，否则会影响Sprint目标。",
        "设计稿已经更新了，可以开始开发。",
        "发现一个边界情况需要处理，详见附件截图。",
        "性能测试结果已出，响应时间在预期范围内。",
        "代码审查通过，准备合并到主分支。",
        "已部署到预发布环境，请大家测试验证。",
        "文档已经同步更新到Wiki上。",
        "建议拆分成两个独立的issue来处理",
        "相关讨论见Slack频道 #project-discussion",
        "单元测试已添加，覆盖率达标",
        "需要产品经理确认需求优先级",
        "生产环境验证通过，可以关闭此工单",
        "这个功能对用户价值很高，建议提升优先级",
        "接口已经开发完成，前端可以开始联调",
        "测试环境已部署，可以在 https://test.example.com 验证",
        "安全审查中发现需要加强输入验证",
        "代码重构完成，复杂度降低了40%",
        "按照最新需求文档更新了实现方案",
        "性能压测TPS达到1200，超过目标值",
        "灰度发布50%流量，监控指标正常",
        "评估了三个技术方案，推荐方案B",
        "客户验收反馈有一条修改意见，详见邮件",
        "数据库迁移脚本已就绪，需要DBA审核",
        "前端组件库更新到最新版本，修复了兼容问题",
        "Postman collection已更新，包含新接口",
        "上线Checklist已填写，等待最终审批",
        "A/B测试数据显示新版方案转化率提升15%",
        "回滚方案已准备，如果出问题5分钟内恢复",
    ]
    total_cmt = 0
    batch = []
    for pid, ids in all_issue_ids.items():
        rng = random.Random(pid * 131 + 7)
        for iid in ids:
            if rng.randint(0, 99) < 50:
                n = 1 + rng.randint(0, 4)
                commenter = users[rng.randint(0, len(users)-1)]
                for c in range(n):
                    text = comment_texts[rng.randint(0, len(comment_texts)-1)]
                    batch.append((iid, text, commenter["id"]))
                    total_cmt += 1
                    if len(batch) >= 200:
                        psycopg2.extras.execute_values(cur, """
                            INSERT INTO comments (issue_id, body, author_id, created_at, updated_at)
                            VALUES %s
                        """, batch, template="(%s, %s, %s, NOW(), NOW())")
                        conn.commit()
                        batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO comments (issue_id, body, author_id, created_at, updated_at)
            VALUES %s
        """, batch, template="(%s, %s, %s, NOW(), NOW())")
        conn.commit()
    print(f"  Created {total_cmt} comments")

    # ========================================================================
    # 8. ACTIVITIES (3-12 per issue) -- has BaseModel
    # ========================================================================
    print("\n--- Step 7: Creating Activities ---")
    activity_verbs = ["created", "updated", "commented", "assigned", "state_changed",
                      "priority_changed", "labeled", "added_to_cycle", "added_to_module", "mentioned"]
    total_act = 0
    batch = []
    for pid, entries in all_new_entries.items():
        states = states_by_project.get(pid, [])
        labels = labels_by_project.get(pid, [])
        state_map = {s["id"]: s for s in states}
        rng = random.Random(pid * 271 + 13)
        for entry in entries:
            if len(entry) < 11:
                continue
            seq, state_id, start_date_str, target_date, completed_at, prio, title, _dh, _ds, type_tag, issue_id = entry
            n = 3 + rng.randint(0, 10)
            for k in range(n):
                verb = activity_verbs[rng.randint(0, len(activity_verbs)-1)]
                actor = users[rng.randint(0, len(users)-1)]
                field, old_val, new_val = None, None, None
                if verb == "created":
                    field = "issue"
                    new_val = title[:80]
                elif verb == "state_changed":
                    field = "state"
                    old_st = states[rng.randint(0, len(states)-1)]
                    old_val = old_st["name"]
                    new_val = state_map.get(state_id, states[0])["name"]
                elif verb == "priority_changed":
                    field = "priority"
                    old_val = priorities[rng.randint(0, len(priorities)-1)]
                    new_val = prio
                elif verb == "assigned":
                    field = "assignee"
                    new_val = users[rng.randint(0, len(users)-1)]["display_name"]
                elif verb == "labeled":
                    field = "label"
                    new_val = labels[rng.randint(0, len(labels)-1)]["name"] if labels else "tag"
                elif verb == "commented":
                    field = "comment"
                    new_val = "添加了评论"
                elif verb == "updated":
                    field = "description"
                    new_val = "更新了描述"
                elif verb == "mentioned":
                    field = "mention"
                    new_val = f"@{users[rng.randint(0, len(users)-1)]['username']}"

                batch.append((issue_id, verb, field, old_val, new_val, actor["id"]))
                total_act += 1
                if len(batch) >= 500:
                    psycopg2.extras.execute_values(cur, """
                        INSERT INTO issue_activities (issue_id, verb, field, old_value, new_value, actor_id, created_at, updated_at)
                        VALUES %s
                    """, batch, template="(%s, %s, %s, %s, %s, %s, NOW(), NOW())")
                    conn.commit()
                    batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO issue_activities (issue_id, verb, field, old_value, new_value, actor_id, created_at, updated_at)
            VALUES %s
        """, batch, template="(%s, %s, %s, %s, %s, %s, NOW(), NOW())")
        conn.commit()
    print(f"  Created {total_act} activities")

    # ========================================================================
    # 9. TIME TRACKS (30% of in-progress/done issues) -- has BaseModel
    # ========================================================================
    print("\n--- Step 8: Creating Time Tracks ---")
    track_descs = ["需求分析", "技术方案设计", "编码实现", "Code Review修改", "自测",
                   "修复测试问题", "文档编写", "部署验证", "性能调优", "架构评审",
                   "数据迁移", "联调测试", "bug修复", "回归测试", "上线支持"]
    total_tt = 0
    batch = []
    for pid, entries in all_new_entries.items():
        states = states_by_project.get(pid, [])
        group_map = {s["id"]: s["group"] for s in states}
        rng = random.Random(pid * 711 + 37)
        for entry in entries:
            if len(entry) < 11:
                continue
            seq, state_id, start_date_str, target_date, completed_at, prio, title, _dh, _ds, type_tag, issue_id = entry
            grp = group_map.get(state_id, "")
            if rng.randint(0, 99) < 30 and grp in ("started", "completed"):
                n = 1 + rng.randint(0, 7)
                try:
                    base_dt = datetime.date.fromisoformat(start_date_str)
                except (ValueError, TypeError):
                    base_dt = datetime.date.today()
                base_dt = datetime.datetime.combine(base_dt, datetime.time(9, 0))
                for t in range(n):
                    ts = base_dt + datetime.timedelta(days=t * rng.randint(0, 2), hours=rng.randint(0, 8))
                    dur_min = rng.randint(15, 480)
                    dur = dur_min * 60  # runtime stores seconds
                    ended = ts + datetime.timedelta(seconds=dur)
                    tracker = users[rng.randint(0, len(users)-1)]
                    tdesc = track_descs[rng.randint(0, len(track_descs)-1)]
                    batch.append((issue_id, tracker["id"], ts.isoformat(), ended.isoformat(), dur, tdesc))
                    total_tt += 1
                    if len(batch) >= 300:
                        psycopg2.extras.execute_values(cur, """
                            INSERT INTO time_tracks (issue_id, user_id, started_at, ended_at, duration, description, created_at, updated_at)
                            VALUES %s
                        """, batch, template="(%s, %s, %s, %s, %s, %s, NOW(), NOW())")
                        conn.commit()
                        batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO time_tracks (issue_id, user_id, started_at, ended_at, duration, description, created_at, updated_at)
            VALUES %s
        """, batch, template="(%s, %s, %s, %s, %s, %s, NOW(), NOW())")
        conn.commit()
    print(f"  Created {total_tt} time tracks")

    # ========================================================================
    # 10. ISSUE RELATIONS -- has BaseModel
    # ========================================================================
    print("\n--- Step 9: Creating Issue Relations ---")
    total_ir = 0
    batch = []
    for ws in workspaces:
        ws_id = ws["id"]
        rel_types = rel_types_by_workspace.get(ws_id, [])
        if not rel_types:
            continue
        cur.execute("SELECT id FROM issues WHERE workspace_id = %s ORDER BY id", (ws_id,))
        ws_issues = [r["id"] for r in cur.fetchall()]
        if len(ws_issues) < 10:
            continue

        # Map relation type names to IDs
        block_id = related_id = parent_id = None
        for rt in rel_types:
            cur.execute("SELECT name FROM relation_types WHERE id = %s", (rt["id"],))
            row = cur.fetchone()
            name = row["name"]
            if "阻塞" in name:
                block_id = rt["id"]
            elif "关联" in name:
                related_id = rt["id"]
            elif "父子" in name:
                parent_id = rt["id"]

        rng = random.Random(ws_id * 911 + 41)
        n = min(50, len(ws_issues) // 2)
        for j in range(n):
            a = rng.randint(0, len(ws_issues)-1)
            b = rng.randint(0, len(ws_issues)-1)
            if a == b:
                continue
            rr = rng.randint(0, 99)
            if rr < 50 and block_id:
                rid = block_id
            elif rr < 85 and related_id:
                rid = related_id
            elif parent_id:
                rid = parent_id
            else:
                continue
            batch.append((ws_issues[a], ws_issues[b], rid))
            total_ir += 1
            if len(batch) >= 100:
                psycopg2.extras.execute_values(cur, """
                    INSERT INTO issue_relations (issue_id, related_issue_id, relation_type_id, created_at, updated_at)
                    VALUES %s
                    ON CONFLICT DO NOTHING
                """, batch, template="(%s, %s, %s, NOW(), NOW())")
                conn.commit()
                batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO issue_relations (issue_id, related_issue_id, relation_type_id, created_at, updated_at)
            VALUES %s
            ON CONFLICT DO NOTHING
        """, batch, template="(%s, %s, %s, NOW(), NOW())")
        conn.commit()
    print(f"  Created {total_ir} issue relations")

    # ========================================================================
    # 11. RELEASE ISSUES -- join table, no timestamps
    # ========================================================================
    print("\n--- Step 10: Creating Release-Issue Links ---")
    cur.execute("SELECT id, project_id FROM releases ORDER BY id")
    releases = cur.fetchall()
    rels_by_project = {}
    for r in releases:
        rels_by_project.setdefault(r["project_id"], []).append(r["id"])

    total_ri = 0
    batch = []
    for pid, rel_ids in rels_by_project.items():
        cur.execute("SELECT id FROM issues WHERE project_id = %s", (pid,))
        proj_issues = [r["id"] for r in cur.fetchall()]
        if not proj_issues:
            continue
        rng = random.Random(pid * 811 + 53)
        for rel_id in rel_ids:
            n = rng.randint(5, 15)
            shuffled = list(proj_issues)
            rng.shuffle(shuffled)
            for j in range(min(n, len(shuffled))):
                batch.append((rel_id, shuffled[j]))
                total_ri += 1
                if len(batch) >= 200:
                    psycopg2.extras.execute_values(cur, """
                        INSERT INTO release_issues (release_id, issue_id)
                        VALUES %s
                        ON CONFLICT DO NOTHING
                    """, batch, template="(%s, %s)")
                    conn.commit()
                    batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO release_issues (release_id, issue_id)
            VALUES %s
            ON CONFLICT DO NOTHING
        """, batch, template="(%s, %s)")
        conn.commit()
    print(f"  Created {total_ri} release-issue links")

    # ========================================================================
    # 12. PROJECT UPDATES -- has created_at only (no updated_at)
    # ========================================================================
    print("\n--- Step 11: Creating Project Updates ---")
    update_statuses = ["on_track", "at_risk", "off_track"]
    update_contents = [
        "本周完成了核心模块的开发和自测，代码审查通过，准备进入测试阶段。",
        "由于第三方API变更，部分功能需要调整实现方案，预计延期2天。",
        "Sprint回顾会议已召开，团队整体进度正常，质量指标达标。",
        "新成员完成入职培训，已开始承担开发任务。",
        "生产环境监控指标一切正常，本周无重大事件。",
        "需求变更已确认，对应的开发任务已更新并重新排期。",
        "性能压测通过，TPS达到预期目标的120%。",
        "安全扫描发现的2个中危漏洞已修复，待发布到生产。",
        "客户验收测试通过，准备下周部署到生产环境。",
        "技术债务清理工作已完成30%，剩余项已排入下个Sprint。",
        "本次Sprint交付了5个Feature和3个关键Bug修复，Sprint目标达成。",
        "灰度发布10%流量，关键指标正常，准备全量发布。",
        "客户反馈了3个改进需求，已纳入下个迭代计划。",
        "系统平均响应时间从200ms降至120ms，性能优化效果显著。",
        "新版UI设计评审通过，前端团队已开始组件开发。",
    ]
    total_pu = 0
    batch = []
    for proj in projects:
        rng = random.Random(proj["id"] * 1011 + 59)
        n = 10 + rng.randint(0, 10)
        for u in range(n):
            author = users[rng.randint(0, len(users)-1)]
            status = update_statuses[rng.randint(0, 2)]
            content = update_contents[rng.randint(0, len(update_contents)-1)]
            batch.append((proj["id"], author["id"], status, content))
            total_pu += 1
            if len(batch) >= 100:
                psycopg2.extras.execute_values(cur, """
                    INSERT INTO project_updates (project_id, author_id, status, content, created_at)
                    VALUES %s
                """, batch, template="(%s, %s, %s, %s, NOW())")
                conn.commit()
                batch = []
    if batch:
        psycopg2.extras.execute_values(cur, """
            INSERT INTO project_updates (project_id, author_id, status, content, created_at)
            VALUES %s
        """, batch, template="(%s, %s, %s, %s, NOW())")
        conn.commit()
    print(f"  Created {total_pu} project updates")

    # ========================================================================
    # 13. SUB-ISSUES (~8% get parent relationship)
    # ========================================================================
    print("\n--- Step 12: Creating Sub-Issues ---")
    total_sub = 0
    for pid, ids in all_issue_ids.items():
        if len(ids) < 10:
            continue
        rng = random.Random(pid * 1111 + 67)
        n = max(5, int(len(ids) * 0.08))
        for _ in range(n):
            ci = rng.randint(0, len(ids)-1)
            pi = rng.randint(0, len(ids)-1)
            if ci == pi:
                continue
            cur.execute("UPDATE issues SET parent_id = %s, updated_at = NOW() WHERE id = %s AND parent_id IS NULL",
                       (ids[pi], ids[ci]))
            if cur.rowcount > 0:
                total_sub += 1
        conn.commit()
    print(f"  Created {total_sub} sub-issue parent links")

    # ========================================================================
    # 14. NOTIFICATIONS for admin -- has BaseModel
    # ========================================================================
    print("\n--- Step 13: Creating Notifications ---")
    notifications = [
        ("数据初始化完成", f"海量测试数据已生成：3个工作空间、10个项目、~{total_issues_created}个新工作项。", "success", "high"),
        ("系统数据概览", f"当前数据库中包含完整的活动记录、评论和工时数据，可全面体验所有功能。", "info", "medium"),
        ("欢迎使用 ReqMango 增强版", "您的项目管理平台已配备充足的测试数据，请开始探索。", "info", "medium"),
        ("工作日开始提醒", "今天有多个工作项即将到期，请及时处理。", "reminder", "medium"),
        ("Sprint 进度预警", "当前 Sprint 剩余 2 天，部分任务推进较慢，建议关注。", "warning", "high"),
        ("团队动态周报已生成", "上周团队完成了多个工作项，代码审查通过率良好。", "info", "medium"),
        ("新版本需求评审邀请", "V2.5 版本需求评审定于周三下午 3 点，请提前准备评审材料。", "reminder", "high"),
    ]
    psycopg2.extras.execute_values(cur, """
        INSERT INTO notifications (title, message, type, priority, recipient_id, created_at, updated_at)
        VALUES %s
    """, [(n[0], n[1], n[2], n[3], admin_id) for n in notifications],
       template="(%s, %s, %s, %s, %s, NOW(), NOW())")
    conn.commit()
    print(f"  Created {len(notifications)} notifications")

    # ========================================================================
    # FINAL SUMMARY
    # ========================================================================
    print("\n" + "=" * 60)
    print("=== 数据初始化完成 ===")
    cur.execute("SELECT COUNT(*) as cnt FROM issues")
    total_issues = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM comments")
    total_c = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM issue_activities")
    total_a = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM time_tracks")
    total_t = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM issue_assignees")
    total_as = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM issue_labels")
    total_il2 = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM issue_relations")
    total_ir2 = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM project_updates")
    total_pu2 = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM issue_cycles")
    total_ic2 = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM module_issues")
    total_mi2 = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM release_issues")
    total_ri2 = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM search_templates")
    total_st = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM saved_views")
    total_sv = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM pages")
    total_pg = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM releases")
    total_rel = cur.fetchone()["cnt"]
    cur.execute("SELECT COUNT(*) as cnt FROM initiatives")
    total_in = cur.fetchone()["cnt"]

    print(f"  Users:              {len(users)}")
    print(f"  Workspaces:         {len(workspaces)}")
    print(f"  Projects:           {len(projects)}")
    print(f"  Issues:             {total_issues}")
    print(f"  Comments:           {total_c}")
    print(f"  Activities:         {total_a}")
    print(f"  Assignees:          {total_as}")
    print(f"  Issue Labels:       {total_il2}")
    print(f"  Issue Cycles:       {total_ic2}")
    print(f"  Module Issues:      {total_mi2}")
    print(f"  Time Tracks:        {total_t}")
    print(f"  Issue Relations:    {total_ir2}")
    print(f"  Release Issues:     {total_ri2}")
    print(f"  Project Updates:    {total_pu2}")
    print(f"  Search Templates:   {total_st}")
    print(f"  Saved Views:        {total_sv}")
    print(f"  Pages:              {total_pg}")
    print(f"  Releases:           {total_rel}")
    print(f"  Initiatives:        {total_in}")
    print("=" * 60)

    print("\nPer-project breakdown:")
    for p in projects:
        cur.execute("SELECT COUNT(*) as cnt FROM issues WHERE project_id = %s", (p["id"],))
        cnt = cur.fetchone()["cnt"]
        cur.execute("SELECT COUNT(*) as cnt FROM comments c JOIN issues i ON c.issue_id = i.id WHERE i.project_id = %s", (p["id"],))
        cc = cur.fetchone()["cnt"]
        cur.execute("SELECT COUNT(*) as cnt FROM time_tracks t JOIN issues i ON t.issue_id = i.id WHERE i.project_id = %s", (p["id"],))
        tc = cur.fetchone()["cnt"]
        print(f"  {p['identifier']:12s}: {cnt:5d} issues | {cc:5d} comments | {tc:5d} time-tracks")

    cur.close()
    conn.close()
    print("\n[SUCCESS] 海量数据初始化成功！")
    print(f"服务访问：http://localhost:5173 | 登录：admin@reqmango.com / demo1234")

if __name__ == "__main__":
    main()
