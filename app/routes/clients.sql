-- Aggregate clients with their contacts to avoid PocketBase's N+1 expand on client_contacts_via_client
SELECT
  c.id,
  c.name,
  COALESCE(
    (
      SELECT json_group_array(
        json_object(
          'id', cc.id,
          'given_name', cc.given_name,
          'surname', cc.surname,
          'email', cc.email
        )
      )
      FROM client_contacts cc
      WHERE cc.client = c.id
    ),
    '[]'
  ) AS contacts_json
FROM clients c
WHERE ({:id} IS NULL OR {:id} = '' OR c.id = {:id})
ORDER BY c.name; 