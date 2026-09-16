SELECT review_status, COUNT(*)
FROM science_assets
GROUP BY review_status
ORDER BY review_status;

SELECT COUNT(*) AS published_without_question
FROM science_assets sa
WHERE sa.review_status = 'published'
  AND NOT EXISTS (
    SELECT 1 FROM questions q
    WHERE q.kp_id = sa.kp_id AND q.code = 'recognize'
  );
