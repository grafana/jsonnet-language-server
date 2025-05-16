local k = import 'ksonnet-util/kausal.libsonnet';

{
  my_deploy: k.apps.v1.deployment.new(
    'my-deploy',
    1,
    k.core.v1.container.new('test', 'alpine:latest'),
  ),
}

{
  my_deploy+: k.util.resourcesRequestsMixin('100m', '100Mi'),
}
