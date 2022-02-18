local imported = import 'goto-import-nested3.libsonnet';
local obj = imported.api.v1.obj;

{
  my_obj:
    obj.new('test') +
    obj.withAttribute('hello') +
    obj.nestedSelf.withAttribute('hello'),

}
