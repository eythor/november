-- SQLite Schema for FHIR Healthcare Data
-- This schema represents FHIR resources with their relationships

-- Core entity tables
CREATE TABLE IF NOT EXISTS patients (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Patient',
    given_name TEXT,
    family_name TEXT,
    prefix TEXT,
    gender TEXT,
    birth_date DATE,
    marital_status TEXT,
    phone TEXT,
    address_line TEXT,
    city TEXT,
    state TEXT,
    postal_code TEXT,
    country TEXT,
    ssn TEXT,
    drivers_license TEXT,
    passport TEXT,
    race TEXT,
    ethnicity TEXT,
    birth_place TEXT,
    mothers_maiden_name TEXT,
    language TEXT,
    raw_json TEXT
);

CREATE TABLE IF NOT EXISTS practitioners (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Practitioner',
    given_name TEXT,
    family_name TEXT,
    prefix TEXT,
    gender TEXT,
    address_line TEXT,
    city TEXT,
    state TEXT,
    postal_code TEXT,
    country TEXT,
    raw_json TEXT
);

CREATE TABLE IF NOT EXISTS organizations (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Organization',
    name TEXT,
    type TEXT,
    phone TEXT,
    address_line TEXT,
    city TEXT,
    state TEXT,
    postal_code TEXT,
    country TEXT,
    raw_json TEXT
);

CREATE TABLE IF NOT EXISTS locations (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Location',
    name TEXT,
    type TEXT,
    address_line TEXT,
    city TEXT,
    state TEXT,
    postal_code TEXT,
    country TEXT,
    latitude REAL,
    longitude REAL,
    raw_json TEXT
);

-- Clinical tables
CREATE TABLE IF NOT EXISTS encounters (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Encounter',
    status TEXT,
    class TEXT,
    type_code TEXT,
    type_display TEXT,
    patient_id TEXT,
    practitioner_id TEXT,
    organization_id TEXT,
    location_id TEXT,
    start_datetime DATETIME,
    end_datetime DATETIME,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (practitioner_id) REFERENCES practitioners(id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
    FOREIGN KEY (location_id) REFERENCES locations(id)
);

CREATE TABLE IF NOT EXISTS conditions (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Condition',
    clinical_status TEXT,
    verification_status TEXT,
    category TEXT,
    code TEXT,
    display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    onset_datetime DATETIME,
    recorded_date DATE,
    abatement_datetime DATETIME,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id)
);

CREATE TABLE IF NOT EXISTS observations (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Observation',
    status TEXT,
    category TEXT,
    code TEXT,
    display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    practitioner_id TEXT,
    effective_datetime DATETIME,
    issued_datetime DATETIME,
    value_quantity REAL,
    value_unit TEXT,
    value_string TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (practitioner_id) REFERENCES practitioners(id)
);

CREATE TABLE IF NOT EXISTS procedures (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Procedure',
    status TEXT,
    code TEXT,
    display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    performer_id TEXT,
    performed_datetime DATETIME,
    performed_period_start DATETIME,
    performed_period_end DATETIME,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (performer_id) REFERENCES practitioners(id)
);

CREATE TABLE IF NOT EXISTS immunizations (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Immunization',
    status TEXT,
    vaccine_code TEXT,
    vaccine_display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    performer_id TEXT,
    occurrence_datetime DATETIME,
    primary_source BOOLEAN,
    lot_number TEXT,
    expiration_date DATE,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (performer_id) REFERENCES practitioners(id)
);

CREATE TABLE IF NOT EXISTS allergy_intolerances (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'AllergyIntolerance',
    clinical_status TEXT,
    verification_status TEXT,
    type TEXT,
    category TEXT,
    criticality TEXT,
    code TEXT,
    display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    recorded_date DATE,
    recorder_id TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (recorder_id) REFERENCES practitioners(id)
);

-- Medication tables
CREATE TABLE IF NOT EXISTS medications (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Medication',
    code TEXT,
    display TEXT,
    form TEXT,
    raw_json TEXT
);

CREATE TABLE IF NOT EXISTS medication_requests (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'MedicationRequest',
    status TEXT,
    intent TEXT,
    medication_id TEXT,
    medication_code TEXT,
    medication_display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    requester_id TEXT,
    authored_on DATETIME,
    dosage_text TEXT,
    dispense_quantity REAL,
    dispense_unit TEXT,
    raw_json TEXT,
    FOREIGN KEY (medication_id) REFERENCES medications(id),
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (requester_id) REFERENCES practitioners(id)
);

CREATE TABLE IF NOT EXISTS medication_administrations (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'MedicationAdministration',
    status TEXT,
    medication_id TEXT,
    medication_code TEXT,
    medication_display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    performer_id TEXT,
    effective_datetime DATETIME,
    dosage_text TEXT,
    dosage_quantity REAL,
    dosage_unit TEXT,
    raw_json TEXT,
    FOREIGN KEY (medication_id) REFERENCES medications(id),
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (performer_id) REFERENCES practitioners(id)
);

-- Diagnostic tables
CREATE TABLE IF NOT EXISTS diagnostic_reports (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'DiagnosticReport',
    status TEXT,
    category TEXT,
    code TEXT,
    display TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    performer_id TEXT,
    effective_datetime DATETIME,
    issued_datetime DATETIME,
    conclusion TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (performer_id) REFERENCES practitioners(id)
);

CREATE TABLE IF NOT EXISTS imaging_studies (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'ImagingStudy',
    status TEXT,
    modality TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    started_datetime DATETIME,
    number_of_series INTEGER,
    number_of_instances INTEGER,
    procedure_code TEXT,
    procedure_display TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id)
);

-- Administrative tables
CREATE TABLE IF NOT EXISTS care_plans (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'CarePlan',
    status TEXT,
    intent TEXT,
    category TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    author_id TEXT,
    created_datetime DATETIME,
    period_start DATETIME,
    period_end DATETIME,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (author_id) REFERENCES practitioners(id)
);

CREATE TABLE IF NOT EXISTS care_teams (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'CareTeam',
    status TEXT,
    category TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    period_start DATETIME,
    period_end DATETIME,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id)
);

-- Financial tables
CREATE TABLE IF NOT EXISTS claims (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Claim',
    status TEXT,
    type TEXT,
    use TEXT,
    patient_id TEXT,
    provider_id TEXT,
    priority TEXT,
    created_datetime DATETIME,
    billable_period_start DATETIME,
    billable_period_end DATETIME,
    total_amount DECIMAL(10,2),
    currency TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (provider_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS explanation_of_benefits (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'ExplanationOfBenefit',
    status TEXT,
    type TEXT,
    use TEXT,
    patient_id TEXT,
    provider_id TEXT,
    claim_id TEXT,
    created_datetime DATETIME,
    billable_period_start DATETIME,
    billable_period_end DATETIME,
    total_amount DECIMAL(10,2),
    payment_amount DECIMAL(10,2),
    currency TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (provider_id) REFERENCES organizations(id),
    FOREIGN KEY (claim_id) REFERENCES claims(id)
);

-- Other resources
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Device',
    status TEXT,
    type_code TEXT,
    type_display TEXT,
    manufacturer TEXT,
    model TEXT,
    serial_number TEXT,
    patient_id TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id)
);

CREATE TABLE IF NOT EXISTS document_references (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'DocumentReference',
    status TEXT,
    type_code TEXT,
    type_display TEXT,
    category TEXT,
    patient_id TEXT,
    encounter_id TEXT,
    author_id TEXT,
    created_datetime DATETIME,
    content_type TEXT,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id),
    FOREIGN KEY (author_id) REFERENCES practitioners(id)
);

CREATE TABLE IF NOT EXISTS practitioner_roles (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'PractitionerRole',
    active BOOLEAN,
    practitioner_id TEXT,
    organization_id TEXT,
    code TEXT,
    specialty TEXT,
    period_start DATETIME,
    period_end DATETIME,
    raw_json TEXT,
    FOREIGN KEY (practitioner_id) REFERENCES practitioners(id),
    FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS supply_deliveries (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'SupplyDelivery',
    status TEXT,
    type_code TEXT,
    type_display TEXT,
    patient_id TEXT,
    supplier_id TEXT,
    occurrence_datetime DATETIME,
    quantity REAL,
    raw_json TEXT,
    FOREIGN KEY (patient_id) REFERENCES patients(id),
    FOREIGN KEY (supplier_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS provenance (
    id TEXT PRIMARY KEY,
    resource_type TEXT DEFAULT 'Provenance',
    target_id TEXT,
    target_type TEXT,
    recorded_datetime DATETIME,
    agent_type TEXT,
    agent_id TEXT,
    raw_json TEXT
);

-- Indexes for common queries
CREATE INDEX idx_encounters_patient ON encounters(patient_id);
CREATE INDEX idx_encounters_date ON encounters(start_datetime);
CREATE INDEX idx_conditions_patient ON conditions(patient_id);
CREATE INDEX idx_observations_patient ON observations(patient_id);
CREATE INDEX idx_observations_encounter ON observations(encounter_id);
CREATE INDEX idx_procedures_patient ON procedures(patient_id);
CREATE INDEX idx_immunizations_patient ON immunizations(patient_id);
CREATE INDEX idx_medication_requests_patient ON medication_requests(patient_id);
CREATE INDEX idx_diagnostic_reports_patient ON diagnostic_reports(patient_id);
CREATE INDEX idx_claims_patient ON claims(patient_id);

-- View for patient summary
CREATE VIEW patient_summary AS
SELECT 
    p.id,
    p.given_name || ' ' || p.family_name as full_name,
    p.gender,
    p.birth_date,
    COUNT(DISTINCT e.id) as encounter_count,
    COUNT(DISTINCT c.id) as condition_count,
    COUNT(DISTINCT o.id) as observation_count,
    COUNT(DISTINCT pr.id) as procedure_count,
    COUNT(DISTINCT i.id) as immunization_count
FROM patients p
LEFT JOIN encounters e ON p.id = e.patient_id
LEFT JOIN conditions c ON p.id = c.patient_id
LEFT JOIN observations o ON p.id = o.patient_id
LEFT JOIN procedures pr ON p.id = pr.patient_id
LEFT JOIN immunizations i ON p.id = i.patient_id
GROUP BY p.id;
