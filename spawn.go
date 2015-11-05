// Copyright 2015 Openprovider Authors. All rights reserved.
// Use of this source code is governed by a license
// that can be found in the LICENSE file.

/*
Package spawn used as a HTTP REST sync service, that makes clustering mode simpler and easier for most
of applications.

What's the idea? There are several applications, which are developed to provide their service
through HTTP Rest API. But you have no idea how to provide failover processing and clustering mode for these
applications, because they are not compatible with, etc. And here we go. The Spawn service will make this
job instead of the whole bunch of services that must be configured and communicating with each other.

Spawn Sync Service
*/
package spawn
