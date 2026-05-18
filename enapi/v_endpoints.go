package enapi

import (
	"context"
	"encoding/json"
	"net/url"
)

// listV is a generic helper for all v/ GET endpoints.
func listV[T any](c *Client, ctx context.Context, path string, opts ...VQueryOption) (*PaginatedResponse[T], error) {
	query := url.Values{}
	for _, opt := range opts {
		opt(query)
	}
	var dest PaginatedResponse[T]
	if err := c.doGet(ctx, path, query, &dest); err != nil {
		return nil, err
	}
	return &dest, nil
}

// ---- Event ----

type EventPersonnel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Event struct {
	ID            string           `json:"id"`
	EventName     string           `json:"event_name"`
	Category      string           `json:"category"`
	StartDatetime Time             `json:"start_datetime"`
	EndDatetime   Time             `json:"end_datetime"`
	Duration      string           `json:"duration"`
	AllDayEvent   IntBool          `json:"all_day_event"`
	Description   string           `json:"description"`
	Station       string           `json:"station"`
	EventType     string           `json:"event_type"`
	LocationName  string           `json:"location_name"`
	Address       string           `json:"address"`
	City          string           `json:"city"`
	State         string           `json:"state"`
	Zip           string           `json:"zip"`
	Notes         string           `json:"notes"`
	Personnel     []EventPersonnel `json:"personnel"`
	CurrentStatus string           `json:"current_status"`
	CreatedAt     Time             `json:"created_at"`
	UpdatedAt     Time             `json:"updated_at"`
}

func (c *Client) ListEvents(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[Event], error) {
	return listV[Event](c, ctx, "/v/events", opts...)
}

// ---- FireBilling ----

type FireBillingVehicleInfo struct {
	ID                       string   `json:"id"`
	DriversName              string   `json:"drivers_name"`
	DriversAddress1          string   `json:"drivers_address_1"`
	DriversAddress2          string   `json:"drivers_address_2"`
	DriversCity              string   `json:"drivers_city"`
	DriversState             string   `json:"drivers_state"`
	DriversZip               string   `json:"drivers_zip"`
	DriversPhone             string   `json:"drivers_phone"`
	DriversInsCo             string   `json:"drivers_ins_co"`
	DriversInsCoPhone        string   `json:"drivers_ins_co_phone"`
	DriversInsAgent          string   `json:"drivers_ins_agent"`
	DriversInsAgencyPhone    string   `json:"drivers_ins_agency_phone"`
	DriversPolicy            string   `json:"drivers_policy"`
	OccupantCount            string   `json:"occupant_count"`
	VehicleMake              string   `json:"vehicle_make"`
	VehicleModel             string   `json:"vehicle_model"`
	VehicleYear              string   `json:"vehicle_year"`
	VehicleColor             string   `json:"vehicle_color"`
	VehicleLicensePlate      string   `json:"vehicle_license_plate"`
	VehicleLicensePlateState string   `json:"vehicle_license_plate_state"`
	VehicleVin               string   `json:"vehicle_vin"`
	SceneProcedures          []string `json:"scene_procedures"`
}

type FireBillingUnitPersonnel struct {
	ID                string `json:"id"`
	CrewMember        string `json:"crew_member"`
	FirstActionTaken  string `json:"first_action_taken"`
	SecondActionTaken string `json:"second_action_taken"`
}

type FireBillingUnit struct {
	ID                string                     `json:"id"`
	Apparatus         string                     `json:"apparatus"`
	FirstActionTaken  string                     `json:"first_action_taken"`
	SecondActionTaken string                     `json:"second_action_taken"`
	Narrative         string                     `json:"narrative"`
	Personnel         []FireBillingUnitPersonnel `json:"personnel"`
}

type FireBillingGearReplaced struct {
	ID                     string `json:"id"`
	GearTypeReplaced       string `json:"gear_type_replaced"`
	TotalGearReplacedCount string `json:"total_gear_replaced_count"`
	GearReplacedComments   string `json:"gear_replaced_comments"`
}

type FireBillingPersonInvolved struct {
	ID                   string `json:"id"`
	BusinessName         string `json:"business_name"`
	Phone                string `json:"phone"`
	FirstName            string `json:"first_name"`
	LastName             string `json:"last_name"`
	Dob                  Time   `json:"dob"`
	Ssn                  string `json:"ssn"`
	DriversLicenseNumber string `json:"drivers_license_number"`
	Designation          string `json:"designation"`
	DesignationOther     string `json:"designation_other"`
	LocationAddress      string `json:"location_address"`
	LocationPrefix       string `json:"location_prefix"`
	LocationStreet       string `json:"location_street"`
	LocationStreetType   string `json:"location_street_type"`
	LocationPoBox        string `json:"location_po_box"`
	LocationApt          string `json:"location_apt"`
	LocationState        string `json:"location_state"`
}

type FireBilling struct {
	ID                        string                      `json:"id"`
	AbsorbentBagsUsed         string                      `json:"absorbent_bags_used"`
	AbsorbentBagsUsedCount    string                      `json:"absorbent_bags_used_count"`
	GallonsFoamUsed           string                      `json:"gallons_foam_used"`
	GallonsFoamUsedCount      string                      `json:"gallons_foam_used_count"`
	AttackLinesPulled         string                      `json:"attack_lines_pulled"`
	LandingZoneUsed           string                      `json:"landing_zone_used"`
	LandingZoneComments       string                      `json:"landing_zone_comments"`
	MotorVehicleFire          string                      `json:"motor_vehicle_fire"`
	MotorVehicleFireComments  string                      `json:"motor_vehicle_fire_comments"`
	RescueExtrication         string                      `json:"rescue_extrication"`
	RescueExtricationComments string                      `json:"rescue_extrication_comments"`
	SceneSafety               string                      `json:"scene_safety"`
	GearCleaned               string                      `json:"gear_cleaned"`
	SetsCleanedCount          string                      `json:"sets_cleaned_count"`
	IncidentNumber            string                      `json:"incident_number"`
	PrimaryActionTaken        string                      `json:"primary_action_taken"`
	AdditionalActionTaken     []string                    `json:"additional_action_taken"`
	Alarm                     Time                        `json:"alarm"`
	OfficerInCharge           string                      `json:"officer_in_charge"`
	LocationState             string                      `json:"location_state"`
	Arrival                   Time                        `json:"arrival"`
	LastUnitCleared           Time                        `json:"last_unit_cleared"`
	CadIncidentNumber         string                      `json:"cad_incident_number"`
	IncidentType              string                      `json:"incident_type"`
	LocationAddress           string                      `json:"location_address"`
	LocationCity              string                      `json:"location_city"`
	LocationZip               string                      `json:"location_zip"`
	LocationLatitude          string                      `json:"location_latitude"`
	LocationLongitude         string                      `json:"location_longitude"`
	Narrative                 string                      `json:"narrative"`
	OwnerPersonInvolved       IntBool                     `json:"owner_person_involved"`
	OwnerBusiness             string                      `json:"owner_business"`
	OwnerPhone                string                      `json:"owner_phone"`
	OwnerFirstName            string                      `json:"owner_first_name"`
	OwnerLastName             string                      `json:"owner_last_name"`
	OwnerDob                  Time                        `json:"owner_dob"`
	OwnerSsn                  string                      `json:"owner_ssn"`
	OwnerDriverLicenseNumber  string                      `json:"owner_driver_license_number"`
	OwnerInsCo                string                      `json:"owner_ins_co"`
	OwnerPolicy               string                      `json:"owner_policy"`
	OwnerAgent                string                      `json:"owner_agent"`
	OwnerAgencyPhone          string                      `json:"owner_agency_phone"`
	OwnerLocationMilepost     string                      `json:"owner_location__milepost"`
	OwnerLocationPrefix       string                      `json:"owner_location_prefix"`
	OwnerLocationStreet       string                      `json:"owner_location_street"`
	OwnerLocationStreetType   string                      `json:"owner_location_street_type"`
	OwnerLocationSuffix       string                      `json:"owner_location_suffix"`
	OwnerLocationPoBox        string                      `json:"owner_location_po_box"`
	OwnerLocationApt          string                      `json:"owner_location_apt"`
	OwnerLocationCity         string                      `json:"owner_location_city"`
	OwnerLocationState        string                      `json:"owner_location_state"`
	OwnerLocationZip          string                      `json:"owner_location_zip"`
	BillableActions           []string                    `json:"billable_actions"`
	VehicleInfo               []FireBillingVehicleInfo    `json:"vehicle_info"`
	Units                     []FireBillingUnit           `json:"units"`
	GearReplaced              []FireBillingGearReplaced   `json:"gear_replaced"`
	PersonsInvolved           []FireBillingPersonInvolved `json:"persons_involved"`
	CurrentStatus             string                      `json:"current_status"`
	CreatedAt                 Time                        `json:"created_at"`
	UpdatedAt                 Time                        `json:"updated_at"`
}

func (c *Client) ListFireBilling(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[FireBilling], error) {
	return listV[FireBilling](c, ctx, "/v/fire-billing", opts...)
}

// ---- Hydrant ----

type HydrantFlowTest struct {
	ID               string `json:"id"`
	FlowTestAt       Time   `json:"flow_test_at"`
	CrewMember       string `json:"crew_member"`
	Apparatus        string `json:"apparatus"`
	TestResult       string `json:"test_result"`
	StaticPressure   string `json:"static_pressure"`
	ResidualPressure string `json:"residual_pressure"`
	TwentyPercent    string `json:"twenty_percent"`
	TenPercent       string `json:"ten_percent"`
	ZeroPercent      string `json:"zero_percent"`
}

type HydrantInspection struct {
	ID            string   `json:"id"`
	Start         Time     `json:"start"`
	End           Time     `json:"end"`
	CrewMember    string   `json:"crew_member"`
	Apparatus     string   `json:"apparatus"`
	AnnualService string   `json:"annual_service"`
	Status        string   `json:"status"`
	NeedsPainted  string   `json:"needs_painted"`
	DefectCodes   []string `json:"defect_codes"`
}

type Hydrant struct {
	ID              string              `json:"id"`
	CurrentStatus   string              `json:"current_status"`
	HydrantClass    string              `json:"hydrant_class"`
	Ports           string              `json:"ports"`
	Ownership       string              `json:"ownership"`
	HydrantDistrict string              `json:"hydrant_district"`
	HydrantZone     string              `json:"hydrant_zone"`
	HydrantType     string              `json:"hydrant_type"`
	HydrantID       string              `json:"hydrant_id"`
	YearInstalled   Time                `json:"year_installed"`
	ValveLocation   string              `json:"valve_location"`
	MainSize        string              `json:"main_size"`
	Address         string              `json:"address"`
	City            string              `json:"city"`
	State           string              `json:"state"`
	Zip             string              `json:"zip"`
	Latitude        string              `json:"latitude"`
	Longitude       string              `json:"longitude"`
	BarrelSize      string              `json:"barrel_size"`
	FlowTest        []HydrantFlowTest   `json:"flow_test"`
	Inspections     []HydrantInspection `json:"inspections"`
	CreatedAt       Time                `json:"created_at"`
	UpdatedAt       Time                `json:"updated_at"`
}

func (c *Client) ListHydrants(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[Hydrant], error) {
	return listV[Hydrant](c, ctx, "/v/hydrants", opts...)
}

// ---- Incident ----

type IncidentUnitPersonnel struct {
	ID                string `json:"id"`
	CrewMember        string `json:"crew_member"`
	FirstActionTaken  string `json:"first_action_taken"`
	SecondActionTaken string `json:"second_action_taken"`
}

type IncidentUnit struct {
	ID                    string                  `json:"id"`
	ApparatusResourceType string                  `json:"apparatus_resource_type"`
	ApparatusResponseMode string                  `json:"apparatus_response_mode"`
	Apparatus             string                  `json:"apparatus"`
	FirstActionTaken      string                  `json:"first_action_taken"`
	SecondActionTaken     string                  `json:"second_action_taken"`
	Narrative             string                  `json:"narrative"`
	ApparatusID           string                  `json:"apparatus_id"`
	CanceledEnroute       Time                    `json:"canceled_enroute"`
	UnitStagingTime       Time                    `json:"unit_staging_time"`
	Dispatch              Time                    `json:"dispatch"`
	Enroute               Time                    `json:"enroute"`
	Arrival               Time                    `json:"arrival"`
	Clear                 Time                    `json:"clear"`
	Personnel             []IncidentUnitPersonnel `json:"personnel"`
}

type Incident struct {
	ID                        string         `json:"id"`
	PropertyUse               string         `json:"property_use"`
	Alarm                     Time           `json:"alarm"`
	DispatchedAs              string         `json:"dispatched_as"`
	Station                   string         `json:"station"`
	CommandEstablished        Time           `json:"command_established"`
	LocationAddressUnit       string         `json:"location_address_unit"`
	Three60EvaluationComplete Time           `json:"360_evaluation_complete"`
	LocationAddressUnitType   string         `json:"location_address_unit_type"`
	FirstWater                Time           `json:"first_water"`
	PrimarySearch             Time           `json:"primary_search"`
	Controlled                Time           `json:"controlled"`
	PreIncidentPropertyValue  FlexString     `json:"pre_incident_property_value"`
	PreIncidentContentsValue  FlexString     `json:"pre_incident_contents_value"`
	TotalPreIncidentValue     FlexString     `json:"total_pre_incident_value"`
	PropertyLossValue         FlexString     `json:"property_loss_value"`
	PropertyLossContentsValue FlexString     `json:"property_loss_contents_value"`
	TotalLoss                 FlexString     `json:"total_loss"`
	Alarms                    string         `json:"alarms"`
	CadNotes                  string         `json:"cad_notes"`
	IncidentNumber            string         `json:"incident_number"`
	IncidentSeries            string         `json:"incident_series"`
	IncidentType              string         `json:"incident_type"`
	CadIncidentNumber         string         `json:"cad_incident_number"`
	LocationLatitude          string         `json:"location_latitude"`
	LocationLongitude         string         `json:"location_longitude"`
	LocationAddress           string         `json:"location_address"`
	LocationState             string         `json:"location_state"`
	MutualAid                 string         `json:"mutual_aid"`
	Psap                      Time           `json:"psap"`
	Enroute                   Time           `json:"enroute"`
	Arrival                   Time           `json:"arrival"`
	UnitLeftScene             Time           `json:"unit_left_scene"`
	LastUnitCleared           Time           `json:"last_unit_cleared"`
	Shift                     string         `json:"shift"`
	District                  string         `json:"district"`
	PrimaryActionTaken        string         `json:"primary_action_taken"`
	AdditionalActionsTaken    []string       `json:"additional_actions_taken"`
	Narrative                 string         `json:"narrative"`
	OfficerInCharge           string         `json:"officer_in_charge"`
	PropertyType              string         `json:"property_type"`
	MutualAidGivenFdid        string         `json:"mutual_aid_given_fdid"`
	MutualAidFdid             string         `json:"mutual_aid_fdid"`
	MutualAidLocationState    string         `json:"mutual_aid_location_state"`
	LocationCity              string         `json:"location_city"`
	MixedPropertyUse          string         `json:"mixed_property_use"`
	LocationZip               string         `json:"location_zip"`
	AdditionalResponders      []string       `json:"additional_responders"`
	MutualAidIncidentNumber   string         `json:"mutual_aid_incident_number"`
	MutualAidPersonnelCount   string         `json:"mutual_aid_personnel_count"`
	MutualAidApparatus        []string       `json:"mutual_aid_apparatus"`
	MutualAidArrival          Time           `json:"mutual_aid_arrival"`
	LocationUnitType          string         `json:"location_unit_type"`
	LocationUnitNumber        string         `json:"location_unit_number"`
	MemberMakingReport        string         `json:"member_making_report"`
	Units                     []IncidentUnit `json:"units"`
	CurrentStatus             string         `json:"current_status"`
	CreatedAt                 Time           `json:"created_at"`
	UpdatedAt                 Time           `json:"updated_at"`
}

func (c *Client) ListIncidents(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[Incident], error) {
	return listV[Incident](c, ctx, "/v/incidents", opts...)
}

// ---- Inspection ----

type InspectionChecklistQuestion struct {
	ID          string `json:"id"`
	Question    string `json:"question"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Code        string `json:"code"`
}

type InspectionChecklistCategory struct {
	ID        string                        `json:"id"`
	Name      string                        `json:"name"`
	Questions []InspectionChecklistQuestion `json:"questions"`
}

type InspectionChecklist struct {
	ID         string                         `json:"id"`
	Name       string                         `json:"name"`
	Categories []InspectionChecklistCategory  `json:"categories"`
}

type Inspection struct {
	ID                           string                `json:"id"`
	PropertyID                   string                `json:"property_id"`
	PropertyName                 string                `json:"property_name"`
	LocationAddress              string                `json:"location_address"`
	LocationCity                 string                `json:"location_city"`
	LocationState                string                `json:"location_state"`
	LocationZip                  string                `json:"location_zip"`
	InspectionNotes              string                `json:"inspection_notes"`
	InspectionID                 string                `json:"inspection_id"`
	InspectionStatus             string                `json:"inspection_status"`
	Inspector                    string                `json:"inspector"`
	CurrentBusinessOrPropertyName string               `json:"current_business_or_property_name"`
	InspectionOrReinspection     string                `json:"inspection_or_reinspection"`
	ScheduledAt                  Time                  `json:"scheduled_at"`
	ArrivedAt                    Time                  `json:"arrived_at"`
	ViolationsToBeRechecked      string                `json:"violations_to_be_rechecked"`
	CompletedAt                  Time                  `json:"completed_at"`
	InspectionType               string                `json:"inspection_type"`
	ViolationsSummary            string                `json:"violations_summary"`
	Checklists                   []InspectionChecklist `json:"checklists"`
	CurrentStatus                string                `json:"current_status"`
	CreatedAt                    Time                  `json:"created_at"`
	UpdatedAt                    Time                  `json:"updated_at"`
}

func (c *Client) ListInspections(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[Inspection], error) {
	return listV[Inspection](c, ctx, "/v/inspections", opts...)
}

// ---- InventoryItem ----

type InventoryItem struct {
	ID                 string `json:"id"`
	AssetTag           string `json:"asset_tag"`
	Make               string `json:"make"`
	Model              string `json:"model"`
	BuildingLocation   string `json:"building_location"`
	InventoryID        string `json:"inventory_id"`
	ItemName           string `json:"item_name"`
	AssignedCrewMember string `json:"assigned_crew_member"`
	Type               string `json:"type"`
	Quantity           string `json:"quantity"`
	Description        string `json:"description"`
	AssignedApparatus  string `json:"assigned_apparatus"`
	AssignedStation    string `json:"assigned_station"`
	CurrentStatus      string `json:"current_status"`
	CreatedAt          Time   `json:"created_at"`
	UpdatedAt          Time   `json:"updated_at"`
}

func (c *Client) ListInventory(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[InventoryItem], error) {
	return listV[InventoryItem](c, ctx, "/v/inventory", opts...)
}

// ---- NerisBilling ----

type NerisBillingPersonInvolved struct {
	ID                          string `json:"id"`
	OwnerFirstName              string `json:"owner_first_name"`
	OwnerLastName               string `json:"owner_last_name"`
	OwnerAddress                string `json:"owner_address"`
	OwnerWorkPhoneNumber        string `json:"owner_work_phone_number"`
	OwnerCellPhone              string `json:"owner_cell_phone"`
	InsuranceCompanyName        string `json:"insurance_company_name"`
	InsuranceCompanyPhoneNumber string `json:"insurance_company_phone_number"`
	InsuranceAgentName          string `json:"insurance_agent_name"`
	InsuranceAgentPhoneNumber   string `json:"insurance_agent_phone_number"`
	PolicyNumber                string `json:"policy_number"`
}

type NerisBillingVehicle struct {
	ID                 string `json:"id"`
	VehicleMake        string `json:"vehicle_make"`
	VehicleModel       string `json:"vehicle_model"`
	VehicleYear        string `json:"vehicle_year"`
	LicensePlateNumber string `json:"license_plate_number"`
	VehicleVin         string `json:"vehicle_vin"`
}

type NerisBillingGearReplaced struct {
	ID                     string `json:"id"`
	GearTypeReplaced       string `json:"gear_type_replaced"`
	TotalGearReplacedCount string `json:"total_gear_replaced_count"`
	GearReplacedComments   string `json:"gear_replaced_comments"`
}

type NerisBilling struct {
	ID                       string                       `json:"id"`
	PsapTime                 Time                         `json:"psap_time"`
	CancelledEnroute         Time                         `json:"cancelled_enroute"`
	OnScene                  Time                         `json:"on_scene"`
	IncidentClearTime        Time                         `json:"incident_clear_time"`
	IncidentNumber           string                       `json:"incident_number"`
	Narrative                string                       `json:"narrative"`
	PrimaryIncidentType      string                       `json:"primary_incident_type"`
	MemberCompletingReport   string                       `json:"member_completing_report"`
	Location                 string                       `json:"location"`
	Station                  string                       `json:"station"`
	AbsorbentBagsUsed        string                       `json:"absorbent_bags_used"`
	AbsorbentBagsUsedCount   string                       `json:"absorbent_bags_used_count"`
	GallonsFoamUsed          string                       `json:"gallons_foam_used"`
	GallonsFoamUsedCount     string                       `json:"gallons_foam_used_count"`
	AttackLinesPulled        string                       `json:"attack_lines_pulled"`
	LandingZoneUsed          string                       `json:"landing_zone_used"`
	LandingZoneComments      string                       `json:"landing_zone_comments"`
	MotorVehicleFire         string                       `json:"motor_vehicle_fire"`
	MotorVehicleFireComments string                       `json:"motor_vehicle_fire_comments"`
	RescueExtrication        string                       `json:"rescue_extrication"`
	RescueExtricationComments string                      `json:"rescue_extrication_comments"`
	SceneSafety              string                       `json:"scene_safety"`
	GearCleaned              string                       `json:"gear_cleaned"`
	SetsCleanedCount         string                       `json:"sets_cleaned_count"`
	PersonsInvolved          []NerisBillingPersonInvolved `json:"persons_involved"`
	Vehicles                 []NerisBillingVehicle        `json:"vehicles"`
	Units                    []json.RawMessage            `json:"units"`
	GearReplaced             []NerisBillingGearReplaced   `json:"gear_replaced"`
	CurrentStatus            string                       `json:"current_status"`
	CreatedAt                Time                         `json:"created_at"`
	UpdatedAt                Time                         `json:"updated_at"`
}

func (c *Client) ListNerisBilling(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[NerisBilling], error) {
	return listV[NerisBilling](c, ctx, "/v/neris-billing", opts...)
}

// ---- NerisIncident ----

type NerisIncidentUnit struct {
	ID                          string   `json:"id"`
	UnitName                    string   `json:"unit_name"`
	UnitResourceType            string   `json:"unit_resource_type"`
	UnitResponseMode            string   `json:"unit_response_mode"`
	UnitNumberOfPersonnel       string   `json:"unit_number_of_personnel"`
	UnitAbleToRespond           string   `json:"unit_able_to_respond"`
	UnitDispatchTime            Time     `json:"unit_dispatch_time"`
	UnitEnrouteTime             Time     `json:"unit_enroute_time"`
	UnitCancelledPriorToArrival string   `json:"unit_cancelled_prior_to_arrival"`
	UnitCancelledEnrouteTime    Time     `json:"unit_cancelled_enroute_time"`
	UnitStagingTime             Time     `json:"unit_staging_time"`
	UnitOnSceneTime             Time     `json:"unit_on_scene_time"`
	UnitClearTime               Time     `json:"unit_clear_time"`
	UnitTurnoutTime             string   `json:"unit_turnout_time"`
	UnitTravelTime              string   `json:"unit_travel_time"`
	UnitActionsAndTactics       []string `json:"unit_actions_and_tactics"`
	UnitNarrative               string   `json:"unit_narrative"`
}

type NerisIncidentPersonnel struct {
	ID               string `json:"id"`
	PersonnelName    string `json:"personnel_name"`
	PersonnelUnit    string `json:"personnel_unit"`
	PersonnelStation string `json:"personnel_station"`
	PersonnelRemarks string `json:"personnel_remarks"`
}

type NerisIncidentMutualAid struct {
	ID                        string `json:"id"`
	MutualAidDepartment       string `json:"mutual_aid_department"`
	MutualAidApparatus        string `json:"mutual_aid_apparatus"`
	MutualAidNumberOfPersonnel string `json:"mutual_aid_number_of_personnel"`
	MutualAidDispatchTime     Time   `json:"mutual_aid_dispatch_time"`
	MutualAidEnrouteTime      Time   `json:"mutual_aid_enroute_time"`
	MutualAidOnSceneTime      Time   `json:"mutual_aid_on_scene_time"`
	MutualAidUnitClearTime    Time   `json:"mutual_aid_unit_clear_time"`
}

type NerisIncident struct {
	ID                                  string                   `json:"id"`
	IncidentPsapTime                    Time                     `json:"incident_psap_time"`
	IncidentDispatchTime                Time                     `json:"incident_dispatch_time"`
	IncidentEnrouteTime                 Time                     `json:"incident_enroute_time"`
	IncidentCancelledEnrouteTime        Time                     `json:"incident_cancelled_enroute_time"`
	IncidentArrivalTime                 Time                     `json:"incident_arrival_time"`
	IncidentSceneStagingTime            Time                     `json:"incident_scene_staging_time"`
	IncidentClearTime                   Time                     `json:"incident_clear_time"`
	MedicalAtPatientTime                Time                     `json:"medical_at_patient_time"`
	MedicalEnrouteToHospital            Time                     `json:"medical_enroute_to_hospital"`
	MedicalArrivalAtHospital            Time                     `json:"medical_arrival_at_hospital"`
	MedicalAgencyTransferTime           Time                     `json:"medical_agency_transfer_time"`
	MedicalFacilityTransferTime         Time                     `json:"medical_facility_transfer_time"`
	MedicalHospitalClearTime            Time                     `json:"medical_hospital_clear_time"`
	FireCommandEstablishedTime          Time                     `json:"fire_command_established_time"`
	Fire360CompletedTime                Time                     `json:"fire_360_completed_time"`
	FirePrimarySearchStartTime          Time                     `json:"fire_primary_search_start_time"`
	FirePrimarySearchCompleteTime       Time                     `json:"fire_primary_search_complete_time"`
	FireWaterOnFireTime                 Time                     `json:"fire_water_on_fire_time"`
	FireUnderControlTime                Time                     `json:"fire_under_control_time"`
	FireKnockedDownTime                 Time                     `json:"fire_knocked_down_time"`
	FireSuppressionCompleteTime         Time                     `json:"fire_suppression_complete_time"`
	NerisIncidentNumber                 string                   `json:"neris_incident_number"`
	CadIncidentNumber                   string                   `json:"cad_incident_number"`
	CadAgencyNumber                     string                   `json:"cad_agency_number"`
	CadNotes                            string                   `json:"cad_notes"`
	IncidentShift                       string                   `json:"incident_shift"`
	IncidentDistrict                    string                   `json:"incident_district"`
	IncidentStation                     string                   `json:"incident_station"`
	IncidentSubdivision                 string                   `json:"incident_subdivision"`
	IncidentAlarms                      string                   `json:"incident_alarms"`
	IncidentDispatchedAs                string                   `json:"incident_dispatched_as"`
	IncidentLocationFormattedAddress    string                   `json:"incident_location_formatted_address"`
	IncidentLocationLatitude            string                   `json:"incident_location_latitude"`
	IncidentLocationLongitude           string                   `json:"incident_location_longitude"`
	IncidentLocationUnitValue           string                   `json:"incident_location_unit_value"`
	IncidentLocationCity                string                   `json:"incident_location_city"`
	IncidentLocationState               string                   `json:"incident_location_state"`
	IncidentLocationZip                 string                   `json:"incident_location_zip"`
	IncidentType                        []string                 `json:"incident_type"`
	PrimaryIncidentType                 string                   `json:"primary_incident_type"`
	IncidentActionsAndTactics           []string                 `json:"incident_actions_and_tactics"`
	IncidentPrimaryLocationType         string                   `json:"incident_primary_location_type"`
	MutualAidGivenOrReceived            string                   `json:"mutual_aid_given_or_received"`
	MutualAidDirection                  string                   `json:"mutual_aid_direction"`
	MutualAidType                       string                   `json:"mutual_aid_type"`
	IncidentAdditionalResponders        []string                 `json:"incident_additional_responders"`
	LocationPropertyPreIncidentValue    string                   `json:"location_property_pre-incident_value"`
	LocationPropertyLosses              string                   `json:"location_property_losses"`
	LocationContentsPreIncidentValue    string                   `json:"location_contents_pre-incident_value"`
	LocationContentsLosses              string                   `json:"location_contents_losses"`
	LocationTotalPreIncidentValue       string                   `json:"location_total_pre-incident_value"`
	LocationTotalDollarLosses           string                   `json:"location_total_dollar_losses"`
	IncidentNarrative                   string                   `json:"incident_narrative"`
	MemberCompletingReport              string                   `json:"member_completing_report"`
	OfficerInCharge                     string                   `json:"officer_in_charge"`
	Units                               []NerisIncidentUnit      `json:"units"`
	Personnel                           []NerisIncidentPersonnel `json:"personnel"`
	MutualAid                           []NerisIncidentMutualAid `json:"mutual_aid"`
	CurrentStatus                       string                   `json:"current_status"`
	CreatedAt                           Time                     `json:"created_at"`
	UpdatedAt                           Time                     `json:"updated_at"`
}

func (c *Client) ListNerisIncidents(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[NerisIncident], error) {
	return listV[NerisIncident](c, ctx, "/v/neris-incidents", opts...)
}

// ---- Property ----

type PropertyContact struct {
	ID            string   `json:"id"`
	ContactType   []string `json:"contact_type"`
	FirstName     string   `json:"first_name"`
	LastName      string   `json:"last_name"`
	Email         string   `json:"email"`
	CallPhone     string   `json:"call_phone"`
	BusinessPhone string   `json:"business_phone"`
	Address       string   `json:"address"`
	City          string   `json:"city"`
	State         string   `json:"state"`
	Zip           string   `json:"zip"`
}

type Property struct {
	ID                    string            `json:"id"`
	TotalSquareFeet       string            `json:"total_square_feet"`
	MaximumOccupantLoad   string            `json:"maximum_occupant_load"`
	NumberOfExits         string            `json:"number_of_exits"`
	YearOfConstruction    string            `json:"year_of_construction"`
	RoofType              string            `json:"roof_type"`
	RoofMaterials         []string          `json:"roof_materials"`
	RoofConstruction      []string          `json:"roof_construction"`
	RiskType              string            `json:"risk_type"`
	BuildingClass         string            `json:"building_class"`
	OccupancyType         string            `json:"occupancy_type"`
	PropertyOwnership     string            `json:"property_ownership"`
	PopulationDensity     string            `json:"population_density"`
	PropertyParcelID      string            `json:"property_parcel_id"`
	GateAccessCode        string            `json:"gate_access_code"`
	AssessedValue         string            `json:"assessed_value"`
	OccupancyID           string            `json:"occupancy_id"`
	OccupancyDistrict     string            `json:"occupancy_district"`
	BusinessPropertyName  string            `json:"business_property_name"`
	Phone                 string            `json:"phone"`
	Fax                   string            `json:"fax"`
	Email                 string            `json:"email"`
	Address               string            `json:"address"`
	City                  string            `json:"city"`
	State                 string            `json:"state"`
	Zip                   string            `json:"zip"`
	Latitude              string            `json:"latitude"`
	Longitude             string            `json:"longitude"`
	Notes                 string            `json:"notes"`
	AnnualInspectionDate  Time              `json:"annual_inspection_date"`
	PrePlanAssignment     string            `json:"pre_plan_assignment"`
	PrePlanPlanner        string            `json:"pre_plan_planner"`
	PrePlanZone           string            `json:"pre_plan_zone"`
	PropertyUse           string            `json:"property_use"`
	BuildingAccess        string            `json:"building_access"`
	StructureType         string            `json:"structure_type"`
	BuildingStatus        string            `json:"building_status"`
	NumberOfUnits         string            `json:"number_of_units"`
	StoriesAboveGrade     string            `json:"stories_above_grade"`
	StoriesBelowGrade     string            `json:"stories_below_grade"`
	Contacts              []PropertyContact `json:"contacts"`
	CurrentStatus         string            `json:"current_status"`
	CreatedAt             Time              `json:"created_at"`
	UpdatedAt             Time              `json:"updated_at"`
}

func (c *Client) ListProperties(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[Property], error) {
	return listV[Property](c, ctx, "/v/properties", opts...)
}

// ---- TrainingRecord ----

type TrainingAttendeeCode struct {
	ID       string `json:"id"`
	Standard string `json:"standard"`
	Category string `json:"category"`
	Code     string `json:"code"`
	Hours    string `json:"hours"`
	Grade    string `json:"grade"`
}

type TrainingAttendee struct {
	ID            string                 `json:"id"`
	AttendeeID    string                 `json:"attendee_id"`
	TrainingCodes []TrainingAttendeeCode `json:"training_codes"`
}

type TrainingInstructor struct {
	ID                string `json:"id"`
	ExternalInstructor bool   `json:"external_instructor"`
	CrewMemberID      string `json:"crew_member_id"`
	Name              string `json:"name"`
	Role              string `json:"role"`
	Agency            string `json:"agency"`
	Notes             string `json:"notes"`
}

type TrainingISOStandard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TrainingCode struct {
	ID       string `json:"id"`
	Standard string `json:"standard"`
	Category string `json:"category"`
	Code     string `json:"code"`
	Hours    string `json:"hours"`
}

type TrainingRecord struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Category      string                `json:"category"`
	StartedAt     Time                  `json:"started_at"`
	EndedAt       Time                  `json:"ended_at"`
	Shift         string                `json:"shift"`
	Objectives    string                `json:"objectives"`
	Narrative     string                `json:"narrative"`
	Attendees     []TrainingAttendee    `json:"attendees"`
	Instructors   []TrainingInstructor  `json:"instructors"`
	IsoStandards  []TrainingISOStandard `json:"iso_standards"`
	TrainingCodes []TrainingCode        `json:"trainig_codes"`
	CurrentStatus string                `json:"current_status"`
	CreatedAt     Time                  `json:"created_at"`
	UpdatedAt     Time                  `json:"updated_at"`
}

func (c *Client) ListTraining(ctx context.Context, opts ...VQueryOption) (*PaginatedResponse[TrainingRecord], error) {
	return listV[TrainingRecord](c, ctx, "/v/training", opts...)
}
