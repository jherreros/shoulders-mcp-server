# Profile overlays

Profile directories contain Kustomize overlays for addon groups under `2-addons/manifests/`.

Flux Kustomizations can point directly at these paths. For example, the `small` profile reconciles `2-addons/profiles/small/helm-releases` instead of `2-addons/manifests/helm-releases`, which keeps the base manifests reusable while making profile-specific deletes and value patches visible in the repo.

For a full non-CLI bootstrap, apply one of the Flux profile overlays:

```bash
kubectl apply -k 2-addons/profiles/small/flux
kubectl apply -k 2-addons/profiles/medium/flux
kubectl apply -k 2-addons/profiles/large/flux
```