package vcd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/go-vcloud-director/govcd"
)

var vappName string = "TestAccVcdVAppVapp"
var vappNameAllocated = "TestAccVcdVAppVappAllocated"

func TestAccVcdVApp_PowerOff(t *testing.T) {
	var vapp govcd.VApp

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	var params = StringMap{
		"Org":               testConfig.VCD.Org,
		"Vdc":               testConfig.VCD.Vdc,
		"EdgeGateway":       testConfig.Networking.EdgeGateway,
		"NetworkName":       "TestAccVcdVAppNet",
		"NetworkName2":      "TestAccVcdVAppNet2",
		"NetworkName3":      "TestAccVcdVAppNet3",
		"Catalog":           testConfig.VCD.Catalog.Name,
		"CatalogItem":       testConfig.VCD.Catalog.Catalogitem,
		"VappName":          vappName,
		"VappNameAllocated": vappNameAllocated,
		"FuncName":          "TestAccCheckVcdVApp_basic",
	}
	configText := templateFill(testAccCheckVcdVApp_basic, params)

	params["FuncName"] = "TestAccCheckVcdVApp_powerOff"

	configTextPoweroff := templateFill(testAccCheckVcdVApp_powerOff, params)
	debugPrintf("#[DEBUG] CONFIGURATION basic: %s\n", configText)
	debugPrintf("#[DEBUG] CONFIGURATION poweroff: %s\n", configTextPoweroff)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdVAppDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppExists("vcd_vapp."+vappName, &vapp),
					testAccCheckVcdVAppAttributes(&vapp),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappName, "name", vappName),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappName, "ip", "10.10.102.160"),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappName, "power_on", "true"),
				),
			},

			resource.TestStep{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappNameAllocated, "name", vappNameAllocated),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappNameAllocated, "ip", "allocated"),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappNameAllocated, "power_on", "true"),
				),
			},

			resource.TestStep{
				Config: configTextPoweroff,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppExists("vcd_vapp."+vappName, &vapp),
					testAccCheckVcdVAppAttributes_off(&vapp),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappName, "name", vappName),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappName, "ip", "10.10.103.160"),
					resource.TestCheckResourceAttr(
						"vcd_vapp."+vappName, "power_on", "false"),
				),
			},
		},
	})
}

func testAccCheckVcdVAppExists(n string, vapp *govcd.VApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no VAPP ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		_, vdc, err := conn.GetOrgAndVdc(testConfig.VCD.Org, testConfig.VCD.Vdc)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, testConfig.VCD.Vdc, testConfig.VCD.Org, err)
		}

		resp, err := vdc.FindVAppByName(rs.Primary.ID)
		if err != nil {
			return err
		}

		*vapp = resp

		return nil
	}
}

func testAccCheckVcdVAppDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_vapp" {
			continue
		}

		_, vdc, err := conn.GetOrgAndVdc(testConfig.VCD.Org, testConfig.VCD.Vdc)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, testConfig.VCD.Vdc, testConfig.VCD.Org, err)
		}

		_, err = vdc.FindVAppByName(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("VPCs still exist")
		}

		return nil
	}

	return nil
}

func testAccCheckVcdVAppAttributes(vapp *govcd.VApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if vapp.VApp.Name != vappName {
			return fmt.Errorf("bad name: %s", vapp.VApp.Name)
		}

		if vapp.VApp.Name != vapp.VApp.Children.VM[0].Name {
			return fmt.Errorf("VApp and VM names do not match. %s != %s",
				vapp.VApp.Name, vapp.VApp.Children.VM[0].Name)
		}

		status, _ := vapp.GetStatus()
		if status != "POWERED_ON" {
			return fmt.Errorf("VApp is not powered on")
		}

		return nil
	}
}

func testAccCheckVcdVAppAttributes_off(vapp *govcd.VApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if vapp.VApp.Name != vappName {
			return fmt.Errorf("bad name: %s", vapp.VApp.Name)
		}

		if vapp.VApp.Name != vapp.VApp.Children.VM[0].Name {
			return fmt.Errorf("VApp and VM names do not match. %s != %s",
				vapp.VApp.Name, vapp.VApp.Children.VM[0].Name)
		}

		status, _ := vapp.GetStatus()
		if status != "POWERED_OFF" {
			return fmt.Errorf("VApp is still powered on")
		}

		return nil
	}
}

const testAccCheckVcdVApp_basic = `
resource "vcd_network_routed" "{{.NetworkName}}" {
  name         = "{{.NetworkName}}"
  org          = "{{.Org}}"
  vdc          = "{{.Vdc}}"
  edge_gateway = "{{.EdgeGateway}}"
  gateway      = "10.10.102.1"

  static_ip_pool {
    start_address = "10.10.102.2"
    end_address   = "10.10.102.254"
  }
}

resource "vcd_network_routed" "{{.NetworkName3}}" {
  name         = "{{.NetworkName3}}"
  org          = "{{.Org}}"
  vdc          = "{{.Vdc}}"
  edge_gateway = "{{.EdgeGateway}}"
  gateway      = "10.10.202.1"

  static_ip_pool {
    start_address = "10.10.202.2"
    end_address   = "10.10.202.254"
  }
}

resource "vcd_vapp" "{{.VappName}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.VappName}}"
  template_name = "{{.CatalogItem}}"
  catalog_name  = "{{.Catalog}}"
  network_name  = "${vcd_network_routed.{{.NetworkName}}.name}"
  memory        = 1024
  cpus          = 1
  ip            = "10.10.102.160"
}

resource "vcd_vapp" "{{.VappNameAllocated}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.VappNameAllocated}}"
  template_name = "{{.CatalogItem}}"
  catalog_name  = "{{.Catalog}}"
  network_name  = "${vcd_network_routed.{{.NetworkName3}}.name}"
  memory        = 1024
  cpus          = 1
  ip            = "allocated"
}
`

const testAccCheckVcdVApp_powerOff = `
resource "vcd_network_routed" "{{.NetworkName2}}" {
  org          = "{{.Org}}"
  vdc          = "{{.Vdc}}"
  name         = "{{.NetworkName2}}"
  edge_gateway = "{{.EdgeGateway}}"
  gateway      = "10.10.103.1"

  static_ip_pool {
    start_address = "10.10.103.2"
    end_address   = "10.10.103.170"
  }

  dhcp_pool {
    start_address = "10.10.103.171"
    end_address   = "10.10.103.254"
  }
}

resource "vcd_vapp" "{{.VappName}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.VappName}}"
  template_name = "{{.CatalogItem}}"
  catalog_name  = "{{.Catalog}}"
  network_name  = "${vcd_network_routed.{{.NetworkName2}}.name}"
  memory        = 1024
  cpus          = 1
  ip            = "10.10.103.160"
  power_on      = false
}
`
