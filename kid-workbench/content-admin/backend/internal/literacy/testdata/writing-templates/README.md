# Offline writing-template fixtures

Source: https://github.com/chanind/hanzi-writer-data (npm version 2.0.1).
Downloaded 2026-09-12. Original strokes and medians preserved; only the local schema, coordinate system and attribution wrapper were added. Character data derives from Make Me a Hanzi and Arphic Technology fonts and is distributed under the accompanying Arphic Public License (ARPHICPL.TXT).

These files are explicit import inputs, never a production fallback. POST a selected JSON file to `/api/v1/literacy/chars/:kpId/writing-template` only when the asset character matches. No network request occurs in the material service or during practice.
