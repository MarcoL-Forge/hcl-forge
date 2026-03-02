# Reusable snippet: EBS volume + attachment
# Injected by hclforge when attach_ebs = true

resource "aws_ebs_volume" "data" {
  availability_zone = "${var.region}a"
  size              = 20
  type              = "gp3"

  tags = {
    Name        = "data-volume"
    Environment = var.environment
  }
}

resource "aws_volume_attachment" "data" {
  device_name = "/dev/sdh"
  volume_id   = aws_ebs_volume.data.id
  instance_id = aws_instance.web.id
}
