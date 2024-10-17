Bugfix: Repair nats-js-kv registry

The registry would always send traffic to only one pod. This is now fixed and load should be spread evenly. Also implements watcher method so the cache can use it.

https://github.com/owncloud/ocis/pull/9726
https://github.com/owncloud/ocis/pull/9662
https://github.com/owncloud/ocis/pull/9656
https://github.com/owncloud/ocis/pull/9654
https://github.com/owncloud/ocis/pull/9620
