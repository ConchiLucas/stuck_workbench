# Handwriting policy `ink-match-v1`

Server-side template coverage matching, not OCR or professional handwriting grading.
It ports the App's 64×64 grid, independent bounding-box normalization, 20 samples per
median, radius-3 user ink, radius-4 template ink and radius-5 proximity. The immutable
thresholds are minimum stroke coverage 0.52, mean coverage 0.62 and precision 0.22.

The initial server policy additionally rejects ink occupying over 0.65 of the grid and
requires each template median to have coverage of at least 0.70 from one individual
user stroke (order independent). Without these constraints the browser algorithm
accepts dense or sparse fields of parallel lines for multi-stroke characters. Thresholds
live in evaluate.go; changes require a new policy ID and must not re-grade saved receipts.

Pointer coordinates are [0,1], downward Y; template medians are 1024-coordinate upward Y.
Timestamps are nonnegative and globally nondecreasing. Limits: 64 strokes, 512 points per
stroke, 8192 total points. Blank ink is a valid failed submission; malformed ink is an error.

Licensed real 山、水、一、的 fixtures come from the material service; ARPHICPL.TXT is retained.
Tests cover complete medians, missing strokes, blank, translation/scaling, diagonal
substitution, dense and sparse scribbles, malformed coordinates and time. They do not
establish discrimination of all similar characters, stroke order, or all acceptable human
handwriting variation. Hints are recorded as `assistance: hinted` by the caller.
