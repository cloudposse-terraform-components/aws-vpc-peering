provider "aws" {
  alias = "accepter"

  region = var.accepter_region

  dynamic "assume_role" {
    # module.iam_roles.terraform_role_arn may be null, in which case do not assume a role.
    for_each = compact([local.accepter_aws_assume_role_arn])
    content {
      role_arn = assume_role.value
    }
  }
}
