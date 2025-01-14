---
title: "28. CS3 API purpose"
date: 2024-02-16T16:39:00+01:00
weight: 28
geekdocRepo: https://github.com/owncloud/ocis
geekdocEditPath: edit/master/docs/ocis/adr
geekdocFilePath: 0028-cs3-api-purpose.md
---

* Status: draft
* Deciders: [@butonic](https://github.com/butonic), [@micbar](https://github.com/micbar), [@dragotin](https://github.com/dragotin)
* Date: 2024-02-21

## Context and Problem Statement

oCIS embeds the reva runtime and uses the [CS3 GRPC APIs](https://buf.build/cs3org-buf/cs3apis) for service communication.
The original idea was to use them as a set of cross-institutional APIs:

> CS3APIs enable creation of easily-accessible and integrated science environments, facilitating
cross-institutional research activities and avoiding fragmented silos based on ad-hoc solutions.

For oCIS we already had the existing WebDAV and OCS APIs that all clients are relying on. Directly using
the CS3 API from web clients would require a proxy and a generated client that would consume precious bandwith.
Furthermore, we introduced the spaces concept which brought in the libregraph API, which is a REST/JSON odata API.

The libregraph API has proven to be easier to extend and use while oCIS was under heavy development than the CS3 API.
Both use code generation to reduce the amount of boilerplate, however extending libregraph with custom properties was
easier than coordingating protocol changes in the CS3 API. Which is to be expected with multiple stakeholders.

Another not well understood aspect of the CS3 APIs is authentication. The reva gateway will authenticate a request and
always return an `x-access-token` to the caller. This JWT may contain sensitive data that other authentication protocols
like OAuth 2.0 or OpenID Connect recommend to remain secret. These JWT might also become too big to handle by external
applications like WOPI.

As a result libregraph has in fact become the API oCIS will use to communicate with all sorts of clients and even in
a federated libregraph scenario for cross-instance communication. oCIS only uses the CS3 API for internal communication.

The question is where do we go from here? What is the purpose of the CS3 API in the context of oCIS?

## Decision Drivers <!-- optional -->

## Considered Options

* Leave it as it is
* Evolve as internal API
* libregraph GRPC API

## Decision Outcome

Chosen option: *@butonic: first evolve as internal API to support all libregraph features, then fix authentication and make it available as alternative to REST/JSON ... that would be my personal choice*?

### Positive Consequences:

* TODO

### Negative Consequences:

* TODO

## Pros and Cons of the Options <!-- optional -->

### Leave it as it is

We do not allow external access to the CS3 GRPC endpoints. We need to document that we do not intend to expose the CS3 gateway.

- Bad, current status quo
- Bad, not the intention of the CS3 API
- Bad, still requires coordination with all stakeholders to change it
- Bad, code generation overhead

### Evolve as internal API

We iterate on the CS3 API to add some of the missing parameters that would allow an easier adaptation of the libregraph API, e.g.:
* a share should be able to represent multiple granted permission sets on a resource
* all kinds of listings should allow pagination, sorting and filtering
* the share manager should be made aware of spaces and allow registering spaces to fit the idea of an actively managed registry of spaces
* the gateway might become a set of middlewares
The details will have to be discussed in subsequent ADRs. We can follow the GRPC/protobuf tooling used in https://github.com/cs3org/cs3apis/ but
we should introduce a new version or create a repo in the libregraph organization to freely iterate on the changes.

+ Good, libregraph is easier to use and explore as a developer than the CS3 because you can look it what all the clients do
+ Good, less coordination with other stakeholders so we can iterate more quickly
- Bad, effectively a disconnect from the CS3 community
- Bad, still not the intention of the CS3 API
- Bad, code generation overhead

### libregraph GRPC API

+ Good, GRPC/protobuf is more efficient than the REST/JSON
+ Good, a cross-instance or cross-institutional API is actually the intention of the CS3 API
- Bad, effectively a disconnect from the current CS3 community
- Bad, changing the CS3 api in radical ways will break any existing applications relying on the current CS3 API *@butonic: TODO list any other then the wopiserver*



## Links <!-- optional -->

* Blog post introducing the CS3 APIs: [Increasing interoperability for research clouds: CS3APIs for connecting sync&share storage, applications and science environments](https://cs3mesh4eosc.eu/node/103)

