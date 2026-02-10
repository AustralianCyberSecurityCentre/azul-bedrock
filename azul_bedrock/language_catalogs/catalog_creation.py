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
