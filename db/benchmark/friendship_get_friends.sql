SELECT p.avatar_id, p.id, up.first_name, up.last_name, ua.username, m.link,
       f.status, f.created_at, f.updated_at
FROM (
  SELECT f.created_at, f.updated_at, status,
         CASE
           WHEN requester_id = 5000 THEN addressee_id
           WHEN addressee_id = 5000 THEN requester_id
         END AS friend
  FROM friendship f
  WHERE 5000 IN (requester_id, addressee_id)
    AND status = 'accepted'
) AS f
JOIN profile p ON p.id = friend
JOIN user_profile up ON up.profile_id = friend
JOIN user_account ua ON up.user_account_id = ua.id
LEFT JOIN media m ON p.avatar_id = m.id AND (m.mime_type LIKE 'image/%' OR m.mime_type = 'image')
ORDER BY p.id ASC;
