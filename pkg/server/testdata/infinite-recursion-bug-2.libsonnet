local drone = import './infinite-recursion-bug-3.libsonnet';
{
  pipeline:
    drone.pipeline.docker {
      new(name):
        super.new(name)
        + super.clone.withRetries(3)
        + super.clone.withDepth(10000)
        + super.platform.withArch('amd64')
        + super.withImagePullSecrets(['dockerconfigjson']),
    },

  step:
    drone.pipeline.docker.step,
}
