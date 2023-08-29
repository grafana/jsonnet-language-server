local drone = import './goto-infinite-recursion-bug-2.libsonnet';
{
  steps: drone.step.withCommands([
    'blabla',
  ]),
}
