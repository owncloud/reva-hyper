Bugfix: Fix grouped email notifications store key

Interval and user id is now separated by `_` (key schema: `${INTERVAL}_${USER_ID}`).

https://github.com/owncloud/ocis/pull/10873
