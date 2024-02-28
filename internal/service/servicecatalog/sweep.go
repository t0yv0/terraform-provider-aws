// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package servicecatalog

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep/awsv1"
)

func RegisterSweepers() {
	resource.AddTestSweepers("aws_servicecatalog_budget_resource_association", &resource.Sweeper{
		Name:         "aws_servicecatalog_budget_resource_association",
		Dependencies: []string{},
		F:            sweepBudgetResourceAssociations,
	})

	resource.AddTestSweepers("aws_servicecatalog_constraint", &resource.Sweeper{
		Name:         "aws_servicecatalog_constraint",
		Dependencies: []string{},
		F:            sweepConstraints,
	})

	resource.AddTestSweepers("aws_servicecatalog_principal_portfolio_association", &resource.Sweeper{
		Name:         "aws_servicecatalog_principal_portfolio_association",
		Dependencies: []string{},
		F:            sweepPrincipalPortfolioAssociations,
	})

	resource.AddTestSweepers("aws_servicecatalog_product_portfolio_association", &resource.Sweeper{
		Name:         "aws_servicecatalog_product_portfolio_association",
		Dependencies: []string{},
		F:            sweepProductPortfolioAssociations,
	})

	resource.AddTestSweepers("aws_servicecatalog_product", &resource.Sweeper{
		Name: "aws_servicecatalog_product",
		Dependencies: []string{
			"aws_servicecatalog_provisioning_artifact",
		},
		F: sweepProducts,
	})

	resource.AddTestSweepers("aws_servicecatalog_provisioned_product", &resource.Sweeper{
		Name:         "aws_servicecatalog_provisioned_product",
		Dependencies: []string{},
		F:            sweepProvisionedProducts,
	})

	resource.AddTestSweepers("aws_servicecatalog_provisioning_artifact", &resource.Sweeper{
		Name:         "aws_servicecatalog_provisioning_artifact",
		Dependencies: []string{},
		F:            sweepProvisioningArtifacts,
	})

	resource.AddTestSweepers("aws_servicecatalog_service_action", &resource.Sweeper{
		Name:         "aws_servicecatalog_service_action",
		Dependencies: []string{},
		F:            sweepServiceActions,
	})

	resource.AddTestSweepers("aws_servicecatalog_tag_option_resource_association", &resource.Sweeper{
		Name:         "aws_servicecatalog_tag_option_resource_association",
		Dependencies: []string{},
		F:            sweepTagOptionResourceAssociations,
	})

	resource.AddTestSweepers("aws_servicecatalog_tag_option", &resource.Sweeper{
		Name:         "aws_servicecatalog_tag_option",
		Dependencies: []string{},
		F:            sweepTagOptions,
	})
}

func sweepBudgetResourceAssociations(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.ListPortfoliosInput{}

	err = conn.ListPortfoliosPages(ctx, input, func(page *servicecatalog.ListPortfoliosOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, port := range page.PortfolioDetails {
			if port == nil {
				continue
			}

			resInput := &servicecatalog.ListBudgetsForResourceInput{
				ResourceId: port.Id,
			}

			err = conn.ListBudgetsForResourcePages(ctx, resInput, func(page *servicecatalog.ListBudgetsForResourceOutput, lastPage bool) bool {
				if page == nil {
					return !lastPage
				}

				for _, budget := range page.Budgets {
					if budget == nil {
						continue
					}

					r := ResourceBudgetResourceAssociation()
					d := r.Data(nil)
					d.SetId(BudgetResourceAssociationID(aws.ToString(budget.BudgetName), aws.ToString(port.Id)))

					sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
				}

				return !lastPage
			})
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Budget Resource (Portfolio) Associations for %s: %w", region, err))
	}

	prodInput := &servicecatalog.SearchProductsAsAdminInput{}

	err = conn.SearchProductsAsAdminPages(ctx, prodInput, func(page *servicecatalog.SearchProductsAsAdminOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, pvd := range page.ProductViewDetails {
			if pvd == nil || pvd.ProductViewSummary == nil {
				continue
			}

			resInput := &servicecatalog.ListBudgetsForResourceInput{
				ResourceId: pvd.ProductViewSummary.ProductId,
			}

			err = conn.ListBudgetsForResourcePages(ctx, resInput, func(page *servicecatalog.ListBudgetsForResourceOutput, lastPage bool) bool {
				if page == nil {
					return !lastPage
				}

				for _, budget := range page.Budgets {
					if budget == nil {
						continue
					}

					r := ResourceBudgetResourceAssociation()
					d := r.Data(nil)
					d.SetId(BudgetResourceAssociationID(aws.ToString(budget.BudgetName), aws.ToString(pvd.ProductViewSummary.ProductId)))

					sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
				}

				return !lastPage
			})
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Budget Resource (Product) Associations for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Budget Resource Associations for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Budget Resource Associations sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepConstraints(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	// no paginator or list operation for constraints directly, have to list portfolios and constraints of portfolios

	input := &servicecatalog.ListPortfoliosInput{}

	err = conn.ListPortfoliosPages(ctx, input, func(page *servicecatalog.ListPortfoliosOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, detail := range page.PortfolioDetails {
			if detail == nil {
				continue
			}

			constraintInput := &servicecatalog.ListConstraintsForPortfolioInput{
				PortfolioId: detail.Id,
			}

			err = conn.ListConstraintsForPortfolioPages(ctx, constraintInput, func(page *servicecatalog.ListConstraintsForPortfolioOutput, lastPage bool) bool {
				if page == nil {
					return !lastPage
				}

				for _, detail := range page.ConstraintDetails {
					if detail == nil {
						continue
					}

					r := ResourceConstraint()
					d := r.Data(nil)
					d.SetId(aws.ToString(detail.ConstraintId))

					sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
				}

				return !lastPage
			})
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Constraints for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Constraints for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Constraints sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepPrincipalPortfolioAssociations(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.ListPortfoliosInput{}

	err = conn.ListPortfoliosPages(ctx, input, func(page *servicecatalog.ListPortfoliosOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, detail := range page.PortfolioDetails {
			if detail == nil {
				continue
			}

			pInput := &servicecatalog.ListPrincipalsForPortfolioInput{
				PortfolioId: detail.Id,
			}

			err = conn.ListPrincipalsForPortfolioPages(ctx, pInput, func(page *servicecatalog.ListPrincipalsForPortfolioOutput, lastPage bool) bool {
				if page == nil {
					return !lastPage
				}

				for _, principal := range page.Principals {
					if principal == nil {
						continue
					}

					r := ResourcePrincipalPortfolioAssociation()
					d := r.Data(nil)
					d.SetId(PrincipalPortfolioAssociationCreateResourceID(AcceptLanguageEnglish, aws.ToString(principal.PrincipalARN), aws.ToString(detail.Id), aws.ToString(principal.PrincipalType)))

					sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
				}

				return !lastPage
			})

			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("error listing Service Catalog Portfolios for Principals %s: %w", region, err))
				continue
			}
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Principal Portfolio Associations for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Principal Portfolio Associations for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Principal Portfolio Associations sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepProductPortfolioAssociations(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	// no paginator or list operation for associations directly, have to list products and associations of products

	input := &servicecatalog.SearchProductsAsAdminInput{}

	err = conn.SearchProductsAsAdminPages(ctx, input, func(page *servicecatalog.SearchProductsAsAdminOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, detail := range page.ProductViewDetails {
			if detail == nil {
				continue
			}

			productARN, err := arn.Parse(aws.ToString(detail.ProductARN))

			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("error parsing Product ARN for %s: %w", aws.ToString(detail.ProductARN), err))
				continue
			}

			// arn:aws:catalog:us-west-2:187416307283:product/prod-t5thhvquxw2x2

			resourceParts := strings.SplitN(productARN.Resource, "/", 2)

			if len(resourceParts) != 2 {
				errs = multierror.Append(errs, fmt.Errorf("error parsing Product ARN resource for %s: %w", aws.ToString(detail.ProductARN), err))
				continue
			}

			productID := resourceParts[1]

			assocInput := &servicecatalog.ListPortfoliosForProductInput{
				ProductId: aws.String(productID),
			}

			err = conn.ListPortfoliosForProductPages(ctx, assocInput, func(page *servicecatalog.ListPortfoliosForProductOutput, lastPage bool) bool {
				if page == nil {
					return !lastPage
				}

				for _, detail := range page.PortfolioDetails {
					if detail == nil {
						continue
					}

					r := ResourceProductPortfolioAssociation()
					d := r.Data(nil)
					d.SetId(ProductPortfolioAssociationCreateID(AcceptLanguageEnglish, aws.ToString(detail.Id), productID))

					sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
				}

				return !lastPage
			})

			if err != nil {
				errs = multierror.Append(errs, fmt.Errorf("error listing Service Catalog Portfolios for Products %s: %w", region, err))
				continue
			}
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Product Portfolio Associations for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Product Portfolio Associations for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Product Portfolio Associations sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepProducts(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.SearchProductsAsAdminInput{}

	err = conn.SearchProductsAsAdminPages(ctx, input, func(page *servicecatalog.SearchProductsAsAdminOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, pvd := range page.ProductViewDetails {
			if pvd == nil || pvd.ProductViewSummary == nil {
				continue
			}

			id := aws.ToString(pvd.ProductViewSummary.ProductId)

			r := ResourceProduct()
			d := r.Data(nil)
			d.SetId(id)

			sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Products for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Products for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Products sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepProvisionedProducts(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.SearchProvisionedProductsInput{
		AccessLevelFilter: &types.AccessLevelFilter{
			Key:   aws.String(servicecatalog.AccessLevelFilterKeyAccount),
			Value: aws.String("self"), // only supported value
		},
	}

	err = conn.SearchProvisionedProductsPages(ctx, input, func(page *servicecatalog.SearchProvisionedProductsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, detail := range page.ProvisionedProducts {
			if detail == nil {
				continue
			}

			r := ResourceProvisionedProduct()
			d := r.Data(nil)
			d.SetId(aws.ToString(detail.Id))
			d.Set("ignore_errors", true)

			sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Provisioned Products for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Provisioned Products for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Provisioned Products sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepProvisioningArtifacts(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.SearchProductsAsAdminInput{}

	err = conn.SearchProductsAsAdminPages(ctx, input, func(page *servicecatalog.SearchProductsAsAdminOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, pvd := range page.ProductViewDetails {
			if pvd == nil || pvd.ProductViewSummary == nil || pvd.ProductViewSummary.ProductId == nil {
				continue
			}

			productID := aws.ToString(pvd.ProductViewSummary.ProductId)

			artInput := &servicecatalog.ListProvisioningArtifactsInput{
				ProductId: aws.String(productID),
			}

			// there's no paginator for ListProvisioningArtifacts
			for {
				output, err := conn.ListProvisioningArtifacts(ctx, artInput)

				if err != nil {
					errs = multierror.Append(errs, fmt.Errorf("error listing Service Catalog Provisioning Artifacts for product (%s): %w", productID, err))
					break
				}

				for _, pad := range output.ProvisioningArtifactDetails {
					r := ResourceProvisioningArtifact()
					d := r.Data(nil)

					d.SetId(ProvisioningArtifactID(aws.ToString(pad.Id), productID))

					sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
				}

				/*
					// Currently, the API has no token field on input (AWS oops)
					if aws.ToString(output.NextPageToken) == "" {
						break
					}

					artInput.NextPageToken = output.NextPageToken
				*/
				break
			}
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Provisioning Artifacts for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Provisioning Artifacts for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Provisioning Artifacts sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepServiceActions(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.ListServiceActionsInput{}

	err = conn.ListServiceActionsPages(ctx, input, func(page *servicecatalog.ListServiceActionsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, sas := range page.ServiceActionSummaries {
			if sas == nil {
				continue
			}

			id := aws.ToString(sas.Id)

			r := ResourceServiceAction()
			d := r.Data(nil)
			d.SetId(id)

			sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
		}

		return !lastPage
	})

	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Service Actions for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Service Actions for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Service Actions sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepTagOptionResourceAssociations(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.ListTagOptionsInput{}

	err = conn.ListTagOptionsPages(ctx, input, func(page *servicecatalog.ListTagOptionsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, tag := range page.TagOptionDetails {
			if tag == nil {
				continue
			}

			resInput := &servicecatalog.ListResourcesForTagOptionInput{
				TagOptionId: tag.Id,
			}

			err = conn.ListResourcesForTagOptionPages(ctx, resInput, func(page *servicecatalog.ListResourcesForTagOptionOutput, lastPage bool) bool {
				if page == nil {
					return !lastPage
				}

				for _, resource := range page.ResourceDetails {
					if resource == nil {
						continue
					}

					r := ResourceTagOptionResourceAssociation()
					d := r.Data(nil)
					d.SetId(aws.ToString(resource.Id))

					sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
				}

				return !lastPage
			})
		}

		return !lastPage
	})

	if tfawserr.ErrCodeEquals(err, servicecatalog.ErrCodeTagOptionNotMigratedException) {
		log.Printf("[WARN] Skipping Service Catalog Tag Option Resource Associations sweep for %s: %s", region, err)
		return nil
	}
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Tag Option Resource Associations for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Tag Option Resource Associations for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Tag Option Resource Associations sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}

func sweepTagOptions(region string) error {
	ctx := sweep.Context(region)
	client, err := sweep.SharedRegionalSweepClient(ctx, region)

	if err != nil {
		return fmt.Errorf("error getting client: %w", err)
	}

	conn := client.ServiceCatalogClient(ctx)
	sweepResources := make([]sweep.Sweepable, 0)
	var errs *multierror.Error

	input := &servicecatalog.ListTagOptionsInput{}

	err = conn.ListTagOptionsPages(ctx, input, func(page *servicecatalog.ListTagOptionsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, tod := range page.TagOptionDetails {
			if tod == nil {
				continue
			}

			id := aws.ToString(tod.Id)

			r := ResourceTagOption()
			d := r.Data(nil)
			d.SetId(id)

			sweepResources = append(sweepResources, sweep.NewSweepResource(r, d, client))
		}

		return !lastPage
	})

	if tfawserr.ErrCodeEquals(err, servicecatalog.ErrCodeTagOptionNotMigratedException) {
		log.Printf("[WARN] Skipping Service Catalog Tag Options sweep for %s: %s", region, err)
		return nil
	}
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error describing Service Catalog Tag Options for %s: %w", region, err))
	}

	if err = sweep.SweepOrchestrator(ctx, sweepResources); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("error sweeping Service Catalog Tag Options for %s: %w", region, err))
	}

	if awsv1.SkipSweepError(errs.ErrorOrNil()) {
		log.Printf("[WARN] Skipping Service Catalog Tag Options sweep for %s: %s", region, errs)
		return nil
	}

	return errs.ErrorOrNil()
}
