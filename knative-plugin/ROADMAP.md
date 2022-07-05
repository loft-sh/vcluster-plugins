#### Acceptance roadmap for knative serving:

##### Service:
Implement following scenarios:
- [x] `Serving` CRD synced with vcluster
- [x] Can create `ksvc` in vcluster
- [x] `ksvc` is synced down to host cluster as expected
- [x] `status` subresource UpSync to virtual object
- [x] `spec.traffic` sync down
- [x] `configuration` `image` sync down creates new `revision`
- [x] Update virtual `ksvc` with 50:50 traffic split and sync down

Add e2e tests for the following scenarios
- [x] Setup e2e testing for knative services
- [x] `Serving` CRD synced with vcluster
- [x] Can create `ksvc` in vcluster
- [x] `ksvc` is synced down to host cluster as expected
- [x] Test `status` subresource UpSync to virtual object
- [x] Check if `ksvc` is reachable at the published endpoint
- [ ] Test sync down of `spec.traffic.latestRevision`
- [ ] Verify `spec.traffic` sync down
- [ ] Test `configuration` `image` sync down creates new `revision`
- [ ] Check `100%` traffic for `v1.0.0`
- [ ] Test update virtual `ksvc` with 50:50 traffic split and sync down
- [ ] Check if traffic split actually works at published endpoint
##### Route:

##### Configuration: