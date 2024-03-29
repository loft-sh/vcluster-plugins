#### Acceptance roadmap for knative serving:

##### Service:
Implement following scenarios:
- [x] `Serving` CRD synced with vcluster
- [x] Can create `ksvc` in vcluster
- [x] `ksvc` is synced down to host cluster as expected
- [x] `status` subresource UpSync to virtual object
- [x] `spec.traffic` sync down
- [x] `configuration.template.` `image` sync down creates new `revision`
- [x] Update virtual `ksvc` with 50:50 traffic split and sync down
- [x] `configuration.template.containerConcurrency` sync down
- [x] `configuration.template.timeoutSeconds` sync down

Add e2e tests for the following scenarios
- [x] Setup e2e testing for knative services
- [x] `Serving` CRD synced with vcluster
- [x] Can create `ksvc` in vcluster
- [x] `ksvc` is synced down to host cluster as expected
- [x] Test `status` subresource UpSync to virtual object
- [x] Check if `ksvc` is reachable at the published endpoint
- [x] Test sync down of `spec.traffic.latestRevision`
- [x] Verify `spec.traffic` sync down
- [x] Test `configuration` `image` sync down creates new `revision`
- [x] Check `100%` traffic for `v1.0.0`
- [x] Test update virtual `ksvc` with 50:50 traffic split and sync down
- [ ] Check if traffic split actually works at published endpoint
- [x] check `containerConcurrency` sync
- [x] check `timeoutSeconds` sync

##### Revisions:
- [x] Basic sync up of host to virtual `revision` syncing
- [x] Reverse translation of revision objects
- [ ] Map owner references to correct config object

##### Configuration:
Implement following scenarios:
- [x] SyncUp of `Configuration` objects as a result of physical ksvc CRs
- [x] Update status of `Configuration` objects as and when new `revisions` are created etc.
- [x] Delete `Configuration` objects when physical objects don't exist anymore
- [x] Mapping correct `kservice` as owner
- [x] Reverse Translate expected name for virtual `Configurations`
- [ ] Implement proper reverse name translation of `Configuration` objects to match with the correct virtual `ksvc` objects
- [ ] Link Virtual `Configurations` with the correct owner references to the virtual `ksvc` objects

##### Route:
