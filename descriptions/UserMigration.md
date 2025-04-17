# Migrating Users from MySQL

## The Problem

Users authenticate via OAuth2.0 with Microsoft. We would ideally like to migrate the users and their corresponding profiles and admin_profiles to a new database.

## The Plan

1. We must first setup Microsoft Authentication via OAuth 2.0 in the target database.

2. The MySQL Profiles table, which is dumped as `Profiles.parquet`, contains an `azureId` column. We will populate the `_externalAuths` table as follows:

    - collectionRef: '_pb_users_auth_'
    - created: <timestamp_now>
    - id: <new_pocketbase_id>
    - provider: 'microsoft'
    - providerId: Profiles.parquet.azureId
    - recordRef: Profiles.parquet.pocketbase_uid
    - update: <timestamp_now> (see created)

3. We will populate the `users` table as follows:

    - created: <timestamp_now>
    - email: Profiles.parquet.email
    - emailVisibility: 0
    - id: Profiles.parquet.pocketbase_uid
    - name: join Profiles.parquet.givenName and Profiles.parquet.surname with a space
    - password: ??
    - tokenKey: ??
    - updated: <timestamp_now>
    - username: Profiles.parquet.email part before @ symbol
    - verified: 1

4. We will populate the `profiles` table as follows:

    - created: <timestamp_now>
    - given_name: Profiles.parquet.givenName
    - id: Profiles.parquet.pocketbase_id
    - surname: Profiles.parquet.surname
    - updated: <timestamp_now> (see created)
    - manager: Profiles.parquet.pocketbase_managerUid **TO BE CREATED by joining Profiles to itself on pocketbase_uid?**
    - alternate_manager: Profiles.parquet.pocketbase_alternateManager **TO BE CREATED by joining Profiles to itself on pocketbase_uid**
    - default_division: Profiles.parquet.pocketbase_defaultDivision **TO BE CREATED by dumping divisions (and timetypes from test db)**
    - uid: Profiles.parquet.pocketbase_uid
    - notification_type: 'to be determined, see test database'
