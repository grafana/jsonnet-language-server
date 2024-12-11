local drone = import './infinite-recursion-bug-2.libsonnet';
{
  steps: drone.step.withCommands([
    'blabla',
  ]),
}
