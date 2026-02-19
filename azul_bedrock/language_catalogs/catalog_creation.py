"""This script generates translations of error messages.

It's expected if this file is modified it will be re-run to re-create the translations.
It will also be run in a package build hook.
"""

import os
import pathlib
from functools import cache

from babel.messages.catalog import Catalog as BabCatalog
from babel.messages.pofile import read_po, write_po

from azul_bedrock.exception_enums import ExceptionCodeEnum
from azul_bedrock.settings import LanguageCatalogsEnum

# To add a new translation update _get_all_unchecked_catalogs and create a function that creates a catalog similar
# to the function _get_english_catalog, also add the translation to LanguageCatalogsEnum.


def _create_catalog(translation: dict[str, str]) -> BabCatalog:
    """Create a catalog from a translation."""
    catalog = BabCatalog(
        project="Azul",
        version="10",
        copyright_holder="",
    )

    for msg_id, msg in translation.items():
        catalog.add(msg_id, msg)

    return catalog


def _get_english_catalog() -> BabCatalog:
    """Get all the values ready to write into the english catalog."""
    translation_values = {
        ExceptionCodeEnum.TODO.value: "{message}",
        # DP Get events
        ExceptionCodeEnum.DPGetEventsBadModelType.value: "Invalid model_type '{model_type}' to get events for",
        ExceptionCodeEnum.DPGetEventFailStatusCode.value: "Unable to get events from dispatcher with error: {response_text}",
        ExceptionCodeEnum.DPGetEventNotMultipartForm.value: "Non form data provided by dispatcher with error: {response_text}",
        ExceptionCodeEnum.DPGetEventNotBoundaryFormData.value: "No form data boundary exists with error: {response_text}",
        ExceptionCodeEnum.DPGetEventsMissingResponseInfo.value: "No info filepart was found in the dispatcher response, can't process event!",
        ExceptionCodeEnum.DPGetEventsFilepartMissingFromResponse.value: "No events filepart from dispatcher response (bad loader?): \nresponse_info={response_info}",
        ExceptionCodeEnum.DPGetEventBadResponseWithContent.value: "Invalid response content: {inner_exception}",
        # DP Submit events
        ExceptionCodeEnum.DPSubmitEventsInvalidModel.value: "Invalid model for submit_events: {model}",
        ExceptionCodeEnum.DPSubmitEventMessageTooLarge.value: "An event to submit to dispatcher was too large: {length_of_event}b > {max_message_size}b",
        ExceptionCodeEnum.DPSubmitEventsUnableToContactDP.value: "Unable to contact dispatcher with error {inner_exception}",
        ExceptionCodeEnum.DPSubmitEventsUnableToSubmitEvents.value: "Unable to submit event to dispatcher with error: {response_text}",
        ExceptionCodeEnum.DPSubmitEventsSubmittedEventsWereInvalid.value: "Submitted events were invalid with error: {response_text}",
        # DP Has binary
        ExceptionCodeEnum.DPHasBinaryUnableToRequestFile.value: "Unable to request file.",
        ExceptionCodeEnum.DPHasBinaryNotFound.value: "Binary content not found.",
        ExceptionCodeEnum.DPHasBinaryBadStatusCode.value: "Unable to request file, bad dispatcher status code {status_code}, response: {response_text}",
        # DP Get Binary
        ExceptionCodeEnum.DPGetBinaryOffsetTooLarge.value: "Unable to request file, is offset too large?",
        ExceptionCodeEnum.DPGetBinaryNotFound.value: "Binary content not found.",
        ExceptionCodeEnum.DPGetBinaryBadStatusCode.value: "Unable to request file with bad status code {status_code}, content: {response_text}",
        # DP Get Binary Async
        ExceptionCodeEnum.DPGetBinaryAsyncOffsetTooLarge.value: "Unable to request file, is offset too large?",
        ExceptionCodeEnum.DPGetBinaryAsyncNotFound.value: "Binary content not found.",
        ExceptionCodeEnum.DPGetBinaryAsyncBadStatusCode.value: "Unable to request file with status_code {status_code} and error {response_text}",
        ExceptionCodeEnum.DPAsyncGetBinaryUnexpectedError.value: "Unexpected error when streaming binary, error: {inner_exception}",
        # Other
        ExceptionCodeEnum.ConvertingContentToAsyncIterable.value: "Bad type for _yield_data, Unexpected type when submitting binary {data_type}.",
        ExceptionCodeEnum.InvalidAsyncSubmissionStringBuffer.value: "Unexpected string buffer when submitting binary {data_type}, valid types are binary io.IOBase, UploadFile, AsyncIterable classes and bytes",
        ExceptionCodeEnum.InvalidAsyncSubmissionContentStream.value: "Unexpected type when submitting binary {data_type}, valid types are binary io.IOBase, UploadFile, AsyncIterable classes and bytes",
        # DP Async Submit Binary
        ExceptionCodeEnum.DPAsyncSubmitBinaryUnableToContactDP.value: "Unable to contact dispatcher.",
        ExceptionCodeEnum.DPAsyncSubmitBinaryBadStatusCode.value: "Unable to submit file to dispatcher with status_code {status_code} and error: {response_text}",
        ExceptionCodeEnum.DPAsyncSubmitBinaryInvalidResponseFormat.value: "Error submitting file to dispatcher.",
        # DP Submit Binary
        ExceptionCodeEnum.DPSubmitBinaryInvalidSubmissionStringBuffer.value: "Unexpected string buffer when submitting binary {data_type}, valid types are binary io.IOBase classes and bytes",
        ExceptionCodeEnum.DPSubmitBinaryInvalidSubmissionContentStream.value: "Unexpected type when submitting binary {data_type}, valid types are binary io.IOBase classes and bytes",
        # DP Unable to Submit binary
        ExceptionCodeEnum.DPSubmitBinaryUnableToContactDP.value: "Unable to contact dispatcher with error: {inner_exception}",
        ExceptionCodeEnum.DPSubmitBinaryBadStatusCode.value: "Unable to submit file to dispatcher with status code {status_code} and error {response_text}",
        ExceptionCodeEnum.DPSubmitBinaryInvalidResponseFormat.value: "Error submitting file to dispatcher.",
        # DP Copy Binary
        ExceptionCodeEnum.DPCopyBinaryBadStatusCode.value: "Unable to copy file in dispatcher with status code {status_code} and error {response_text}",
        # DP Delete Binary
        ExceptionCodeEnum.DPDeleteBinaryBadStatusCode.value: "Unable to delete binary from dispatcher with status code {status_code} and error {response_text}",
        # Identify
        ExceptionCodeEnum.IdentifyDosIdentNotDos.value: "Error occurred when attempting to identify the type of a PE file.",
        ExceptionCodeEnum.NoFiletypeIdentificationHappened.value: "Assemblyline type found was '{file_format}' doesn't have a mapping and should",
        # models_network
        ExceptionCodeEnum.FeatureValueEncodingFailure.value: "Cannot encode value {value} with type {value_type}",
        ExceptionCodeEnum.FeatureValueDecodingFailure.value: "Could not decode type {value_type} for value {value}",
        ExceptionCodeEnum.ConvertingStreamToInputEntityFailure.value: "Only content fileinfo can become entity, no content label found on current entity.",
        # models_settings
        ExceptionCodeEnum.ConvertStringToDurationIncorrectNumberOfValuesAfterSplit.value: "provided input '{input_duration}' split into '{length_split_string}' strings it must split into 2, the split was actually {split_string}",
        ExceptionCodeEnum.ConvertStringToDurationInvalidDuration.value: "Invalid duration for expire_events_after duration='{duration}' duration should be an integer value.",
        ExceptionCodeEnum.ConvertStringToDurationInvalidUnitProvided.value: "Invalid unit for expire_events_after unt={unit} valid values are valid_units={valid_units}",
        ### Runner
        # test_template
        ExceptionCodeEnum.TestRunnerExecutionEventTooLarge.value: "event produced by plugin was too large: {length_of_event}b > {max_message_size}b",
        ### Azul-metastore
        # context
        ExceptionCodeEnum.MetastoreContextBadSecurity.value: "security was not valid ({security})",
        ExceptionCodeEnum.MetastoreContextInsufficientPermissionsForWrite.value: "Could not initialise metastore templates",
        # entry
        ExceptionCodeEnum.MetastoreEntryBadInputParameters.value: "When using --no-input, both METASTORE_OPENSEARCH_ADMIN_USERNAME and METASTORE_OPENSEARCH_ADMIN_PASSWORD must be set.",
        # ingestor
        ExceptionCodeEnum.MetastoreIngestorBadStatusDocument.value: "bad status {status_code}\n{content_text}",
        ExceptionCodeEnum.MetastoreIngestorGetDataNetworkError.value: "connection error",
        # settings
        ExceptionCodeEnum.MetastoreSettingsPartitionNotSet.value: "metastore_partition must be set. Recommended to set to dev01, qa01, prod01, etc.",
        ExceptionCodeEnum.MetastoreSettingsFieldsAreMissing.value: "source={source}: missing fields= {fields_missing}",
        ExceptionCodeEnum.MetastoreSettingsExtraFieldsAreMissing.value: "source={source}: has extra fields {fields_extra}",
        # feature
        ExceptionCodeEnum.MetastoreFeatureEnrichmentFailed.value: "failed to parse and enrich feature: {feature} with error message: {inner_exception}",
        # fileformat
        ExceptionCodeEnum.MetastoreFileFormatTooLargeForUnzip.value: "Can't extract file larger than {max_bundled_pre_extract_size}",
        ExceptionCodeEnum.MetastoreFileFormatTooManyFilesToExtract.value: "too many files to extract archive ({number_of_files}/{max_number_of_files})",
        ExceptionCodeEnum.MetastoreFileFormatFileTooLargeToExtract.value: "too many bytes in extracted submission ({total_bytes}/{max_extraction_size})",
        ExceptionCodeEnum.MetastoreFileFormatFilePathTooLong.value: "file path too long ({path_len}/{max_bundled_filename_length}): {first_part_of_path}...",
        ExceptionCodeEnum.MetastoreFileFormatBadPathElevation.value: "file name has bad character sequence (/../): {file_path}",
        ExceptionCodeEnum.MetastoreFileFormatNotAZipFile.value: "not a zip file",
        ExceptionCodeEnum.MetastoreFileFormatUnknownException.value: "Unexpected error of type {error_type}: {error_text}",
        ExceptionCodeEnum.MetastoreFileFormatZipFileRequiresPassword.value: "zip requires password",
        ExceptionCodeEnum.MetastoreFileFormatZipFileBadPasswordProvided.value: "bad zip password",
        ExceptionCodeEnum.MetastoreFileFormatNoFilesExtractedFromZip.value: "no files were extracted from zip",
        ExceptionCodeEnum.MetastoreFileFormatFileTooLargeForUnmalpz.value: "Can't unmalpz file larger than {file_size}",
        # memcache
        ExceptionCodeEnum.MetastoreMemcacheTTLCacheAlreadyCreated.value: "TTL cache already exists {cache_id}",
        ExceptionCodeEnum.MetastoreMemcacheLRUCacheAlreadyCreated.value: "LRU cache already exists {cache_id}",
        # opensearch
        ExceptionCodeEnum.MetastoreOpensearchCantGetUserAccountInner.value: "status_code={status_code}, with response={response_text}",
        ExceptionCodeEnum.MetastoreOpensearchCantGetUserAccount.value: "could not get user account: {inner_exception}",
        # search_data
        ExceptionCodeEnum.MetastoreSearchDataBadCredentials.value: "unrecognised credential format: {credential_format}",
        ExceptionCodeEnum.MetastoreSearchDataMissingOrBadParameters.value: "missing/bad parameter: {inner_exception}",
        # search_query_parser
        ExceptionCodeEnum.MetastoreSearchQueryMissingToken.value: "Token missing start/end",
        ExceptionCodeEnum.MetastoreSearchQueryInvalidUnescapeSequence.value: "Invalid escape character: {character}",
        ExceptionCodeEnum.MetastoreSearchQueryUnterminatedEscapeAtEnd.value: "Unterminated escape at end of string",
        ExceptionCodeEnum.MetastoreSearchQueryNumberOfInputTokens.value: "Input tokens don't match expected length",
        ExceptionCodeEnum.MetastoreSearchQueryNumberExpressionNotInteger.value: "Internal error: RawToken of numeric field is not an integer",
        ExceptionCodeEnum.MetastoreSearchQueryStringFieldNotString.value: "Internal error: RawToken of string field is not an string",
        ExceptionCodeEnum.MetastoreSearchQueryNumberExpressionUnexpectedTokenCount.value: "Unexpected count of tokens when parsing number expression",
        ExceptionCodeEnum.MetastoreSearchQueryInvalidType.value: "Invalid types passed to range expression",
        # search_query
        ExceptionCodeEnum.MetastoreSearchQueryMissingContext.value: "Dynamic evaluation needed for tag searches.",
        ExceptionCodeEnum.MetastoreSearchQueryTagSearchNotOnStringValue.value: "Tag search must be made on a string value.",
        ExceptionCodeEnum.MetastoreSearchQueryTagSearchNotWithEquals.value: "Tag search can only be a literal search.",
        ExceptionCodeEnum.MetastoreSearchQueryTagDoesntExist.value: "tag does not exist in opensearch",
        ExceptionCodeEnum.MetastoreFeatureValueTagNotFound.value: "feature value tag not found",
        ExceptionCodeEnum.MetastoreSearchQueryUnreachableDuringConversion.value: "Hit unreachable during conversion of field",
        ExceptionCodeEnum.MetastoreSearchQuerySearchForImplicitShouldBeLiteral.value: "Search for an implicit field should be a literal, not a range",
        ExceptionCodeEnum.MetastoreSearchQueryAutocompleteParentNotATag.value: "Parent of a node is not a Tag? ({parents})",
        ExceptionCodeEnum.MetastoreSearchQueryInvalidTermKey.value: "Failed to parse term: {inner_exception}",
        # string_filter
        ExceptionCodeEnum.MetastoreAiStringFilterFailure.value: "AI filter failed with error:  {text_content}",
        # tlsh
        ExceptionCodeEnum.MetastoreInvalidTLSHFormat.value: "Invalid TLSH version in: '{tlsh_hash}'",
        ExceptionCodeEnum.MetastoreInvalidTLSHLength.value: "Invalid TLSH length in: '{tlsh_str}'",
        # common/wrapper
        ExceptionCodeEnum.MetastoreOpensearchAuthFailure.value: "Failed to get templates from Opensearch for index: {self.alias}, with error: {e.error}",
        ExceptionCodeEnum.MetastoreOpensearchFailedToCreateIndex.value: "Failure creating index '{index}'",
        ExceptionCodeEnum.MetastoreOpensearchTemplateOldVersion.value: "{index} template ({existing_template_version}) does not match metastore ({new_version}). Consider using a new metastore partition and reindexing data.",
        ExceptionCodeEnum.MetastoreOpensearchKnnMisconfigured.value: "kNN filters only supported for one search term",
        ExceptionCodeEnum.MetastoreOnlyAllowTopLevelBoolOrKnn.value: "Can only have bool in top level query (or within kNN query filter)",
        ExceptionCodeEnum.MetastoreBadSecurityConversionExclude.value: "Bad security provided in security_exclude {security_exclude}. Exception: {inner_exception}",
        ExceptionCodeEnum.MetastoreBadSecurityConversionInclude.value: "Bad security provided in security_include {security_include}. Exception: {inner_exception}",
        ExceptionCodeEnum.MetastoreKnnTooManySearchTerms.value: "kNN filters only supported for one search term",
        ExceptionCodeEnum.MetastoreUnknownDocType.value: "unknown doc type to handle error in {doc_type}",
        # annotation
        ExceptionCodeEnum.MetastoreUnknownAnnotation.value: "unknown annotation {event_type}",
        ExceptionCodeEnum.MetastoreAnnotationBadCharacterInTag.value: "bad characters in tag: {event_tag}",
        ExceptionCodeEnum.MetastoreAnnotationTagTooLong.value: "tag too long: {event_tag}",
        ExceptionCodeEnum.MetastoreAnnotationCommentTooLong.value: "comment too long: {event_comment}",
        # base_encoder
        ExceptionCodeEnum.MetastoreEncoderInvalidTimestamp.value: "invalid timestamp {timestamp}; is not an absolute timestamp",
        ExceptionCodeEnum.MetastoreInvalidPartitionFormatUnits.value: "unknown value index_time_unit={index_time_unit}, should be one of {available_options}",
        # binary2
        ExceptionCodeEnum.MetastoreEncodeErrorMissingTrackingInfo.value: "event is missing tracking information '{item}': {event}",
        ExceptionCodeEnum.MetastoreEncodingMissingSource.value: "Source does not exist: {source_id}",
        ExceptionCodeEnum.MetastoreEncodingInvalidSSDeep.value: "ssdeep could not be parsed {ssdeep}",
        # basic_events
        ExceptionCodeEnum.MetastorePreProcessTooManyStreams.value: "too many streams: {len_streams} > {stream_limit}",
        ExceptionCodeEnum.MetastorePreProcessTooManyFeatures.value: "too many features: {len_features} > {feature_limit}",
        ExceptionCodeEnum.MetastoreEncoderInvalidPluginConfig.value: "plugin config value for '{config_key}' is not json string: {config_value}",
        # age_off
        ExceptionCodeEnum.MetastoreFailedToAgeoffDocs.value: "failed to process ageoff on alias={alias} with expire_events_ms={expire_events_ms} response:\n{resp}",
        # cache
        ExceptionCodeEnum.MetastoreCacheTooManyIds.value: "max counts in one call exceeded",
        # purge
        ExceptionCodeEnum.MetastoreCannotCreatePurgeFolder.value: "to purge data, 'purge_sha256_folder' config option must be set",
        ExceptionCodeEnum.MetastoreMetaDataDeletionFailure.value: "{ret}",
        ExceptionCodeEnum.MetastoreNothingToPurge.value: "nothing to delete",
        # binary_event
        ExceptionCodeEnum.MetastoreBadMappingToOpensearch.value: "Bad mapping object: {value}",
        # binary_feature
        ExceptionCodeEnum.MetastoreInvalidAfterProvided.value: "Invalid after provided '{after}', after must be valid JSON!",
        # binary_find_paginate
        ExceptionCodeEnum.MetastoreFailedToParseTermQuery.value: "Failed to parse term: {inner_exception}",
        ExceptionCodeEnum.MetastoreSha256NotProvidedForFindFamily.value: "Sha256 to search for parent or child binaries for was not set and should have been!",
        # binary_find
        ExceptionCodeEnum.MetastoreBinaryFindTooManyBinariesRequested.value: "too many binaries have been requested max is {requested_binaries}/{max_allowed_binaries}",
        ExceptionCodeEnum.MetastoreBinaryFindFaildToParseSearchTerm.value: "Failed to parse term '{term}': {inner_exception}",
        # binary_read
        ExceptionCodeEnum.MetastoreSha256NotProvidedForFindingStreamRefs.value: "Sha256 was not provided and is required.",
        # binary_similar
        ExceptionCodeEnum.MetastoreLibFuzzyFailedToInitalise.value: "could not find libfuzzy-dev; check that it is installed.",
        ExceptionCodeEnum.MetastoreReadSimilarSSDeepBadFuzzyHash.value: "ssdeep fuzzy hash could not be parsed {fuzzy_hash}",
        # binary_submit_manual
        ExceptionCodeEnum.MetastoreDispatcherRejectedEvents.value: "Dispatcher rejected the submitted events, with response {response}",
        ExceptionCodeEnum.MetastoreSubmissionsCantCreateInsertionEvents.value: "Unable to propagate insert events to metastore",
        # binary_submit
        ExceptionCodeEnum.MetastoreNoSourcesForBinarySubmission.value: "Attempting to upload a binary to no sources!",
        ExceptionCodeEnum.MetastoreBadAugmentedStreamLabel.value: "augmented stream label cannot be 'content'",
        ExceptionCodeEnum.MetastoreInvalidSourceForBinarySubmission.value: "Source does not exist: {source}",
        ExceptionCodeEnum.MetastoreBadSourceReferenceDefinition.value: "{inner_exception}",
        ExceptionCodeEnum.MetastoreUnableToSubmitBinaryEventImmediately.value: "Unable to submit binary event to metastore immediately, {inner_exception}",
        ExceptionCodeEnum.MetastoreBinarySubmitDispatcherRejectedEvent.value: "Dispatcher rejected the submitted events, with response {response}",
        ExceptionCodeEnum.MetastoreBadBinarySubmissionBadSecurityString.value: "Must provide valid security string.",
        ExceptionCodeEnum.MetastoreBadBinarySubmissionUserSecurity.value: "security being applied by the user is greater than the current users security. because user: {inner_exception}",
        ExceptionCodeEnum.MetastoreBadBinarySubmissionSourceOrParent.value: "Must provide source or parent information.",
        ExceptionCodeEnum.MetastoreBadBinarySubmissionParentAndSource.value: "cannot insert binary to source and parent at same time",
        ExceptionCodeEnum.MetastoreBadBinarySubmissionNoSha256Provided.value: "Must supply binary or sha256",
        ExceptionCodeEnum.MetastoreBadBinarySubmissionParentNotFound.value: "Parent Id (sha256) must already exist",
        ExceptionCodeEnum.MetastoreUnableToExtractProvidedArchive.value: "Unable to extract provided archive with error {inner_exception}",
        ExceptionCodeEnum.MetastoreDatalessSubmissionBinaryDoesNotExist.value: "Cannot find existing metadata for entity {sha256}. You will need to supply Azul with the original binary.",
        ExceptionCodeEnum.MetastoreUnableToExtractAnyFiles.value: "Cannot find any files to extract. This may be an unsupported filetype or bad password.",
        # binaries_data
        ExceptionCodeEnum.MetastoreBinaryNotFound.value: "Binary {sha256} not found in Azul.",
        ExceptionCodeEnum.MetastoreBinaryStreamNotFound.value: "The stream for the binary {sha256} was not found in Azul.",
        ExceptionCodeEnum.MetastoreInvalidSha256Provided.value: "One of the provided sha256s '{sha256}' is invalid and the request cannot be processed.",
        ExceptionCodeEnum.MetastoreBulkUnableToDownloadAnyBinaries.value: "No binaries can be downloaded.",
        ExceptionCodeEnum.MetastoreNoBinariesDownloaded.value: "No items in bulk zip file.",
        ExceptionCodeEnum.MetastoreDownloadingBadStreamType.value: "Stream of file tpe '{file_format}' is not allowed for direct download",
        ExceptionCodeEnum.MetastoreInvalidStringsRegexProvided.value: "Invalid regex pattern '{regex}' provided",
        ExceptionCodeEnum.MetastoreInvalidHexPatternProvided.value: "Invalid hex filter provided '{filter}'",
        # binaries_submit
        ExceptionCodeEnum.MetastoreCannotParseTimestampToUTC.value: "a bad timestamp was provided {inner_exception}",
        ExceptionCodeEnum.MetastoreInvalidJson.value: "Bad json for the field {field_name} was provided, value was {field_value}",
        ExceptionCodeEnum.MetastoreAltStreamsLabelsDoesNotMatch.value: "stream labels ({stream_label_count}) must be supplied for all stream data ({stream_data_count})",
        # binaries
        ExceptionCodeEnum.MetastoreFindBinaryInvalidSearch.value: "{inner_exception}",
        ExceptionCodeEnum.MetastorePotentiallyInvalidQueryOption.value: "The following term query key:{opt_s} could not be found: [{invalid_keys_string}]. Either the key:{opt_s} are related to results and temporarily missing or the query is invalid.",
        ExceptionCodeEnum.MetastoreIncludeCousinsInvalidEnum.value: "Provided value for include_cousins={include_cousins} is invalid.",
        ExceptionCodeEnum.MetastoreTagBinaryBadSecurityString.value: "Must provide valid security string.",
        ExceptionCodeEnum.MetastoreTagBinaryInvalidSecurity.value: "security being applied by the user is greater than the current users security. because user: {inner_exception}",
        ExceptionCodeEnum.MetastoreInvalidAnnotationForCreate.value: "{inner_exception}",
        ExceptionCodeEnum.MetastoreCantDeleteTagFromBinary.value: "Cannot delete tag {tag} from the sha256 {sha256}, because it is not found.",
        # features
        ExceptionCodeEnum.MetastoreInvalidDeleteTag.value: "Cannot delete tag {tag} because it doesn't exist.",
        ExceptionCodeEnum.MetastoreNoFeatureValuesFound.value: "No feature values found for feature '{feature}'",
        ExceptionCodeEnum.MetastoreFeaturesInvalidPivotFeatures.value: "No feature values provided, feature values must be a valid list of tuples containing feature_name, feature_value",
        ExceptionCodeEnum.MetastoreNoPluginsInAzul.value: "There are no plugins registered in Azul.",
        ExceptionCodeEnum.MetastoreNoPluginStatusesInAzul.value: "There are no plugin statuses in Azul.",
        ExceptionCodeEnum.MetastorePluginNotInAzul.value: "The requested plugin version does was not found.",
        ExceptionCodeEnum.MetastoreUserNotAllowedToPurge.value: "user '{username}' not superuser",
        ExceptionCodeEnum.MetastoreInvalidTimestampForPurge.value: "The timestamp provided '{timestamp}' has an invalid format.",
        ExceptionCodeEnum.MetastoreInvalidPurgeExceptionApi.value: "{inner_exception}",
        # quick
        ExceptionCodeEnum.MetastoreUserUnauthorized.value: "{inner_exception}",
        ExceptionCodeEnum.MetastoreSetSecurityHeaderUnexpected.value: "Logic error - should be setting response and/or exception!",
        ExceptionCodeEnum.MetastoreUserInfoNotAvailable.value: "user_info is not available on request.state",
        # sources
        ExceptionCodeEnum.MetastoreNoSourcesInAzul.value: "No sources can be found in Azul.",
        ExceptionCodeEnum.MetastoreSourceNotFound.value: "The provided source cannot be found in Azul.",
        ExceptionCodeEnum.MetastoreSourceNoReferences.value: "No references can be found for the provided source.",
        ExceptionCodeEnum.MetastoreSourceSubmissionNoInformationFound.value: "No source information could be found for the given request.",
        ### Security
        # friendly
        ExceptionCodeEnum.SecurityMissingOrigin.value: "has releasability but does not have origin={origin}",
        ExceptionCodeEnum.SecurityClassificationDoesntSupportReleasability.value: "Classifications '{exclusive_string}' have a releasability(s) {inclusive_string} but none of the classifications support releasability.",
        ExceptionCodeEnum.SecurityInvalidReleasabilityGroup.value: "has invalid group in bad_group={bad_group}",
        ExceptionCodeEnum.SecurityInvalidExclusiveGroup.value: "Unregistered security items 'exclusive': {unregistered}",
        ExceptionCodeEnum.SecurityInvalidInclusiveGroup.value: "Unregistered security items 'inclusive': {unregistered}",
        ExceptionCodeEnum.SecurityInvalidMarkingsGroup.value: "Unregistered security items 'markings': {unregistered}",
        ExceptionCodeEnum.SecurityNoClassificationInRawSecurity.value: "no classification in {raw_security}",
        ExceptionCodeEnum.SecurityInvalidGroupsWhileNormalising.value: "invalid groups: {normalised_clean_groups}",
        ExceptionCodeEnum.SecurityInvalidSecurityStringWhileNormalising.value: "{inner_exception}: raw_security={raw_security}",
        ExceptionCodeEnum.SecurityInvalidReleasabilitiesConvertFromLabels.value: "security has invalid groups: {group_diff}",
        ExceptionCodeEnum.SecurityInvalidLabelConvertingFromLabels.value: "security has invalid label {security_label}",
        # security
        ExceptionCodeEnum.SecurityMinRequiredAccessNotFound.value: "minimum required access level ({security_label}) not found in inclusive or exclusive sets",
        ExceptionCodeEnum.SecuritySecurityDefaultNotSet.value: "must set security_default to valid security option",
        ExceptionCodeEnum.SecurityNoCommonSecurityUnviewable.value: "no common inclusive set: {inc}",
        ExceptionCodeEnum.SecurityUserCannotAccessExclusive.value: "User cannot access all {exclusive_difference}",
        ExceptionCodeEnum.SecurityUserCannotAccessInclusive.value: "User cannot access all {inclusive_difference}",
        ExceptionCodeEnum.SecurityUserCannotAccessMarkings.value: "User cannot access all {markings_difference}",
        ExceptionCodeEnum.SecurityUnmatchedLabelsGoingSafeToUnsafe.value: "unmatched safe->unsafe in {labels}",
        ExceptionCodeEnum.SecurityUnmatchedLabelsGoingUnsafeToSafe.value: "unmatched unsafe->safe in {labels}",
        ExceptionCodeEnum.SecurityUserDoesNotHaveMinimumAccess.value: "user does not meet minimum_required_access, missing security labels {missing_labels}",
        # settings
        ExceptionCodeEnum.SecurityConfigLabelsWithExtraSpaces.value: "security labels must not start or end with a space: '{raw_label_name}'",
        ExceptionCodeEnum.SecurityConfigReleasabilitiesMissingRequiredPrefix.value: "All security group labels must be prefixed with '{releasability_prefix}'",
        ExceptionCodeEnum.SecurityConfigLabelDefinedTwice.value: "a security label has been defined twice: {summed}",
        ExceptionCodeEnum.SecurityConfigGroupLabelMustNotHaveSpaces.value: "group labels must not have spaces: '{label}'",
        ExceptionCodeEnum.SecurityConfigMultipleValuesMappedToSameValue.value: "two labels were made safe to the same value: {safe_label}",
        # restapi
        ExceptionCodeEnum.SecurityNormaliseInvalidSecurity.value: "invalid security strings: {inner_exception}",
        ExceptionCodeEnum.SecurityInvalidMaxSecurity.value: "invalid security strings or combination: {inner_exception}",
        ExceptionCodeEnum.SecurityEmptyResultForMaxSecurity.value: "empty result",
        ExceptionCodeEnum.SecurityUserInfoCannotBeAcquired.value: "user_info is not available on request.state",
        ### azul-restapi-server
        # pat
        ExceptionCodeEnum.RestapiAllowedPATAction.value: "user '{username}' not superuser",
    }
    return _create_catalog(translation_values)


def _write_catalog(catalog: BabCatalog, language: LanguageCatalogsEnum):
    """Write a catalog to the catalog directory."""
    catalog_path = _get_catalog_directory().joinpath(language.value)
    with open(catalog_path, "wb") as f:
        write_po(f, catalog)


def _validate_catalog(catalog: BabCatalog, language: LanguageCatalogsEnum):
    """Validate that the catalog has a value for all defined error enums."""
    missing_value_for_keys = []
    for val in ExceptionCodeEnum:
        response_message = catalog.get(val.value)
        if not response_message:
            missing_value_for_keys.append(val.value)

    if missing_value_for_keys:
        raise Exception(
            f"Failed to create translation '{language.value.removesuffix('.po')}' missing value for enums {','.join(missing_value_for_keys)}"
        )


def _get_all_unchecked_catalogs() -> dict[LanguageCatalogsEnum, BabCatalog]:
    """Get a dictionary containing all the catalog translation from strings."""
    languages = {}
    languages[LanguageCatalogsEnum.English] = _get_english_catalog()
    return languages


def _get_catalog_directory() -> pathlib.Path:
    """Get the catalog directory (should be where this python file is)."""
    return pathlib.Path(os.path.dirname(os.path.abspath(__file__)))


@cache
def get_catalog(catalog_language: LanguageCatalogsEnum) -> BabCatalog:
    """Load a catalog from the catalog directory.

    Uses a cache to prevent reloading of the same catalog over and over again.
    """
    catalog_dir = _get_catalog_directory()
    catalog_path = catalog_dir.joinpath(catalog_language.value)
    if not catalog_path.exists():
        raise Exception(
            f"Attempting to load the translation '{catalog_language.value}' but the translation does not exist."
        )
    with open(catalog_path, "rb") as f:
        catalog = read_po(f)

    return catalog


def recreate_catalogs():
    """Re-create all catalogs and write them to disk."""
    for lang, cat in _get_all_unchecked_catalogs().items():
        _validate_catalog(cat, lang)
        _write_catalog(cat, lang)


if __name__ == "__main__":
    recreate_catalogs()
