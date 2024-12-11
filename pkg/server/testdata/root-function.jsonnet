local lib = import 'root-function-lib.libsonnet';
local libResolved = (import 'root-function-lib.libsonnet')('test');

{

  fromImport: (import 'root-function-lib.libsonnet')('test').attribute,
  fromLib: lib('test').attribute,
  fromResolvedLib: libResolved.attribute,

  nestedFromImport: (import 'root-function-lib.libsonnet')('test').nestedFunc('test').nestedAttribute,
  nestedFromLib: lib('test').nestedFunc('test').nestedAttribute,
  nestedFromResolvedLib: libResolved.nestedFunc('test').nestedAttribute,
}
