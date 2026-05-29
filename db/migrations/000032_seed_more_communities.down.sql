DO $$
DECLARE
  target_community_ids BIGINT[];
  target_profile_ids BIGINT[];
  target_post_ids BIGINT[];
BEGIN
  SELECT COALESCE(array_agg(id), ARRAY[]::BIGINT[])
  INTO target_community_ids
  FROM community
  WHERE username = ANY(ARRAY['seeddevlife', 'seedcareer', 'seedsportlife', 'seedrunclub', 'seedanimals', 'seedpets', 'seedcookbook', 'seedcoffee', 'seedtravelers', 'seedwildspace', 'seedreaders', 'seedcinema', 'seedmusicroom', 'seedgamepad', 'seedtabletop', 'seedphotowalk', 'seedartstudio', 'seedvolunteers', 'seedsciencehub', 'seedhomegarden']::TEXT[]);

  SELECT COALESCE(array_agg(profile_id), ARRAY[]::BIGINT[])
  INTO target_profile_ids
  FROM community
  WHERE id = ANY(target_community_ids);

  SELECT COALESCE(array_agg(id), ARRAY[]::BIGINT[])
  INTO target_post_ids
  FROM post
  WHERE community_id = ANY(target_community_ids)
     OR author_id = ANY(target_profile_ids);

  DELETE FROM search_outbox
  WHERE (entity_type = 'community' AND entity_id = ANY(target_community_ids))
     OR (entity_type = 'post' AND entity_id = ANY(target_post_ids));

  DELETE FROM like_record
  WHERE post_id = ANY(target_post_ids)
     OR comment_id IN (SELECT id FROM comment WHERE post_id = ANY(target_post_ids));
  DELETE FROM comment WHERE post_id = ANY(target_post_ids);
  DELETE FROM repost WHERE post_id = ANY(target_post_ids);
  DELETE FROM post WHERE id = ANY(target_post_ids);
  DELETE FROM community_member WHERE community_id = ANY(target_community_ids);

  UPDATE profile
  SET avatar_id = NULL
  WHERE id = ANY(target_profile_ids);

  DELETE FROM community WHERE id = ANY(target_community_ids);
  DELETE FROM media WHERE author_id = ANY(target_profile_ids);
  DELETE FROM profile WHERE id = ANY(target_profile_ids);
END $$;
