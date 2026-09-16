# Shared literacy player

The three React hosts depend on this private local package with `file:` dependencies. React and React DOM are peers; Vite hosts deduplicate both. No router, query cache, child state, database, or grading implementation is part of the player.

`LiteracyPlayer` accepts `question`, `mediaResolver`, `onSubmit`, optional `disabled`, and `mode="practice"|"preview"`. `question.id` starts a fresh interaction session. Images always preserve their original aspect ratio (`object-fit: contain`); missing or failed media display an error rather than generated replacements. Handwriting never renders `stem.text` or receives the standard template.

`onSubmit` receives a stable option ID or normalized 0–1 strokes with monotonic millisecond timestamps and `hintsUsed`. The callback returns `{correct, canRetry?, answerOptionId?}`. The host owns transport, client IDs, authoritative assessment and advancement. Replaying a failed request uses the same response payload. Preview hosts call their own same-origin preview API and do not write learning records.

All practice styling lives in `src/player.css`, scoped to `.literacy-player` with `--lp-*` variables and uniquely prefixed classes. Narrow containers reflow instead of scaling the handwriting canvas. The App keeps its original direct type-practice components and independently organized questions; shared-player use does not require the App to claim backend tasks. Admin previews follow that App presentation.

Preview handwriting does not autoplay when many questions mount together. Practice handwriting autoplays once per question; replay replaces the current audio and unmount stops it. Writing reset clears the canvas, feedback and retry payload.

Run `npm test` here to execute the shared interaction suite using the App's existing Vitest/jsdom installation. The tests live in `literacy-app/src/components/LiteracyPlayer.test.tsx` and cover stable IDs, keyboard audio isolation, retry locks, hidden writing answers, normalized pointer input and clear behavior.
