# Knowledge dashboard spacing implementation

User approved the layout preview and removal of the visible title/slogan row. Keep all metrics, chart types, calendar fields, navigation and data behavior. Work in the existing dirty checkout without touching other services.

- [x] Remove header markup and unused icon; retain an accessible article name. Update the existing overview contract test.
- [x] Refine overview.css in place: 24px outer padding, 16px gaps, 20px panel padding, 96px metrics, 270px charts, responsive 40/28/32 chart columns, 73/27 calendar split with a usable minimum daily width. Calendar rows 80px; narrow screens stack panels and keep all calendar content readable.
- [x] Position daily chart labels relative to the same track and percentage used by each bar; use a shared integer axis with headroom. No data/statistic changes.
- [x] Run frontend tests and production build. Inspect deployed desktop and mobile, calendar selection and chart label alignment.
- [x] Build and replace diagnosis-admin only with --no-deps at 19211. Verify health before exact old-image cleanup; close QA browser and update this record.

Scope note: the existing homepage contains fixed demonstration values and calendar data. This layout-only change preserves that behavior and does not claim to implement real-data aggregation.
