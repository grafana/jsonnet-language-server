local library = import 'multilevel-library-top.libsonnet';
{
  my_item: library.sub1.subsub1.new(name='test'),
}
