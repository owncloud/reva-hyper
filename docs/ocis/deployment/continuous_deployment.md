---
title: "Continuous Deployment"
date: 2020-10-12T14:04:00+01:00
weight: 10
geekdocRepo: https://github.com/owncloud/ocis
geekdocEditPath: edit/master/docs/ocis/deployment
geekdocFilePath: continuous_deployment.md
---

{{< toc >}}

We are continuously deploying the following deployment examples. Every example is deployed in two flavors:

- Master: reflects the current master branch state of oCIS and will be updated with every commit to master
- Rolling: reflects the latest rolling release of oCISt and will be updated with every release
- Production: reflects the latest production release of oCIS and will be updated with every release

The configuration for the continuous deployment can be found in the [oCIS repository](https://github.com/owncloud/ocis/tree/master/deployments/continuous-deployment-config).

# oCIS with Web Office

This deployment is based on our modular [ocis_full Example](ocis_full.md) and uses the default configuration with Collabora Online as the office suite, traefik reverse proxy, cloudimporter and the inbucket mail catching server to showcase the full feature set of oCIS.

Credentials:

- oCIS: see [default demo users]({{< ref "../getting-started#login-to-owncloud-web" >}})

## Master

- oCIS: [ocis.ocis.master.owncloud.works](https://ocis.ocis.rolling.owncloud.works)
- Mail: [mail.ocis.master.owncloud.works](https://mail.ocis.rolling.owncloud.works)

## Rolling Release

- oCIS: [ocis.ocis.rolling.owncloud.works](https://ocis.ocis.rolling.owncloud.works)
- Mail: [mail.ocis.rolling.owncloud.works](https://mail.ocis.rolling.owncloud.works)

## Rolling Release with OnlyOffice

This example is based on the [ocis_full Example](ocis_full.md) and uses the default configuration with OnlyOffice as the office suite.

- oCIS: [ocis.ocis-onlyoffice.rolling.owncloud.works](https://ocis.ocis-wopi.onlyoffice.owncloud.works)

# oCIS and ownCloud Web with both most recent development versions

Credentials:

- oCIS: see [default demo users]({{< ref "../getting-started#login-to-owncloud-web" >}})

## Latest

- oCIS: [ocis.ocis-web.master.owncloud.works](https://ocis.ocis-web.master.owncloud.works)

## Daily

- oCIS: [ocis.ocis-web.daily.owncloud.works](https://ocis.ocis-web.daily.owncloud.works)

# oCIS with Keycloak

Credentials:

- oCIS: see [default demo users]({{< ref "../getting-started#login-to-owncloud-web" >}})
- Keycloak:
  - username: admin
  - password: admin

## Rolling Release

- oCIS: [ocis.ocis-keycloak.rolling.owncloud.works](https://ocis.ocis-keycloak.rolling.owncloud.works)
- Keycloak admin access: [keycloak.ocis-keycloak.rolling.owncloud.works](https://keycloak.ocis-keycloak.rolling.owncloud.works)
- Keycloak account management: [keycloak.ocis-keycloak.rolling.owncloud.works/realms/oCIS/account/#/](https://keycloak.ocis-keycloak.rolling.owncloud.works/realms/oCIS/account/#/)


# oCIS with S3 storage backend (MinIO)

This deployment is based on our modular [ocis_full Example](ocis_full.md) and uses a configuration with Collabora Online as the office suite, traefik reverse proxy, cloudimporter and the inbucket mail catching server to showcase the full feature set of oCIS. In addition to that, we deployed a MinIO S3 storage backend. oCIS stores the data in the S3 server and the metadata on the local disk by using the `s3ng` storage driver.

The MinIO server provides a powerful Web UI for browser-based access to the storage which makes it possible to manage the data stored in the S3 server and understand how different policies and configurations affect the data.

Credentials:

- oCIS: see [default demo users]({{< ref "../getting-started/demo-users/" >}})
- MinIO:
  - access key: ocis
  - secret access key: ocis-secret-key

## Rolling Release

- oCIS: [ocis.ocis-s3.rolling.owncloud.works](https://ocis.ocis-s3.rolling.owncloud.works)
- MinIO: [minio.ocis-s3.rolling.owncloud.works](https://minio.ocis-s3.rolling.owncloud.works)
- Inbucket: [mail.ocis-s3.rolling.owncloud.works](https://mail.ocis-s3.rolling.owncloud.works)

# oCIS with LDAP for users and groups

Credentials:

- oCIS: see [default demo users]({{< ref "../getting-started/demo-users/" >}})
- LDAP admin:
  - username: cn=admin,dc=owncloud,dc=com
  - password: admin

## Rolling Release

- oCIS: [ocis.ocis-ldap.rolling.owncloud.works](https://ocis.ocis-ldap.rolling.owncloud.works)
- LDAP admin: [ldap.ocis-ldap.rolling.owncloud.works](https://ldap.ocis-ldap.rolling.owncloud.works)

