package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"

	"github.com/AustralianCyberSecurityCentre/azul-bedrock/v9/gosrc/models"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

const MILLISECONDS_IN_A_DAY = 1000 * 60 * 60 * 24
const NO_LIFECYCLE_ERROR_CODE = "NoSuchLifecycleConfiguration"

// NOTE - changing this will have negative effects on any existing system that uses these rules.
const EXPIRY_POLICY_PREFIX = "BackupExpiryFor-"

func getRuleId(sourceName string) string {
	return fmt.Sprintf("%s%s", EXPIRY_POLICY_PREFIX, sourceName)
}

/*Remove all the rules that were created on the bucket as part of the setLifecycleForBucket function.*/
func removeLifeCycleForBucket(client *minio.Client, bucket string, sourceConf *models.SourcesConf) error {
	var err error
	currentLifeCycleConfig, err := client.GetBucketLifecycle(context.Background(), bucket)
	if err != nil {
		// If the error is the lifecycle doesn't exist clear it.
		var minioError minio.ErrorResponse
		if errors.As(err, &minioError) {
			if minioError.Code == NO_LIFECYCLE_ERROR_CODE {
				// No lifecycle policy is set and none is required immediately exit.
				return nil
			}
		} else {
			log.Printf("Failed to get the bucket lifecycle policy with the error '%s'\n", err.Error())
			return err
		}
	}

	indexesToRemove := []int{}
	// Identify all the automatically created backup expiry rules.
	for srcName := range sourceConf.Sources {
		ruleId := getRuleId(srcName)
		for i := range currentLifeCycleConfig.Rules {
			if currentLifeCycleConfig.Rules[i].ID == ruleId {
				indexesToRemove = append(indexesToRemove, i)
			}
		}
	}
	// Remove all the automatically created rules.
	newRules := []lifecycle.Rule{}
	for i, rule := range currentLifeCycleConfig.Rules {
		if !slices.Contains(indexesToRemove, i) {
			newRules = append(newRules, rule)
		}
	}
	currentLifeCycleConfig.Rules = newRules

	if len(indexesToRemove) > 0 {
		log.Printf("Lifecycle policy for bucket %s has been disabled removing old policies.\n", bucket)
		err = client.SetBucketLifecycle(context.Background(), bucket, currentLifeCycleConfig)
		if err != nil {
			log.Printf("Failed to remove old lifecycle policy rules with the error '%s'\n", err.Error())
			return err
		}
	}
	log.Println("Automatic AgeOff for backup is disabled.")
	return nil

}

/*Set the lifecycle policy for the Minio storage bucket.*/
func setLifecycleForBucket(client *minio.Client, bucket string, sourceConf *models.SourcesConf) error {
	var err error
	currentLifeCycleConfig, err := client.GetBucketLifecycle(context.Background(), bucket)
	if err != nil {
		// If the error is the lifecycle doesn't exist clear it.
		var minioError minio.ErrorResponse
		if errors.As(err, &minioError) {
			if minioError.Code == NO_LIFECYCLE_ERROR_CODE {
				// No lifecycle policy is set and none is required immediately exit.
				currentLifeCycleConfig = lifecycle.NewConfiguration()
				err = nil
			}
		}
		if err != nil {
			log.Printf("Failed to get the bucket lifecycle policy with the error '%s'\n", err.Error())
			return err
		}
	}

	statusEnabled := "Enabled"

	// If a rule is modified or added an updated set of rules should be sent to minio.
	shouldUpdateLifecycle := false
	for srcName, src := range sourceConf.Sources {
		// One day plus configured expiry (extra day accounts for integer rounding)
		expiryInDays := lifecycle.ExpirationDays(1 + src.ExpireEventsAfterMs/MILLISECONDS_IN_A_DAY)
		ruleId := getRuleId(srcName)
		prefix := fmt.Sprintf("%s/", srcName)

		shouldCreateRule := true
		// Check if the rule already exists.
		for i := range currentLifeCycleConfig.Rules {
			rule := &currentLifeCycleConfig.Rules[i]
			if ruleId == rule.ID {
				// Rule should either be updated or left alone
				shouldCreateRule = false
				if expiryInDays == rule.Expiration.Days &&
					rule.RuleFilter.Prefix == prefix &&
					rule.Status == statusEnabled &&
					rule.Expiration.DeleteAll {
					// The rule to be created already exists.
					fmt.Printf("The source %s in bucket %s is set to expire objects older than %d days.\n", srcName, bucket, expiryInDays)
					break
				} else {
					// The rule to be created doesn't exist and there is an old rule that will need to be removed.
					rule.Expiration.Days = expiryInDays
					rule.Expiration.DeleteAll = true
					rule.RuleFilter = lifecycle.Filter{
						Prefix: prefix,
					}
					rule.Status = statusEnabled
					shouldUpdateLifecycle = true
					log.Printf("Rule %s was updated and will age off %s after %d days", ruleId, srcName, expiryInDays)
				}
			}
		}
		if shouldCreateRule {
			shouldUpdateLifecycle = true
			// Add new rule for creation
			currentLifeCycleConfig.Rules = append(currentLifeCycleConfig.Rules, lifecycle.Rule{
				ID: ruleId,
				Expiration: lifecycle.Expiration{
					Days:      expiryInDays,
					DeleteAll: true,
				},
				Status: statusEnabled,
				RuleFilter: lifecycle.Filter{
					Prefix: prefix,
				},
			})
			log.Printf("Rule %s was created and will age off %s after %d days", ruleId, srcName, expiryInDays)
		}
	}

	if shouldUpdateLifecycle {
		log.Printf("Lifecycle policy for bucket %s was out of date or did not exist, updating now.\n", bucket)
		err = client.SetBucketLifecycle(context.Background(), bucket, currentLifeCycleConfig)
		if err != nil {
			log.Printf("Failed to update the lifecycle policy with the error '%s'\n", err.Error())
			return err
		}
	}
	log.Println("Automatic AgeOff for backup is enabled and has finished being verified and modified.")
	return err
}
