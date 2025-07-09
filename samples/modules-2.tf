module "etl_BarFoo" {
    source = "../"

    foo = module.eks_Foobar.id
    bar = module.ecs_Foobar.id
}
