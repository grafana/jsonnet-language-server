(import 'k8s-libsonnet/main.libsonnet')
+ {
  core+:: { v1+:: { namespace+:: {
    new(name, create=false)::
      // Namespace creation is handled by environments/cluster-resources.
      // https://github.com/grafana/deployment_tools/blob/master/docs/platform/kubernetes/namespaces.md
      if create
      then super.new(name)
      else {},
  } } },
}
