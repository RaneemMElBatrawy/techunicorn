package controllers

import (
	"fmt"
	"log"
	"net/http"
	"techunicorn/models"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateDoctor struct {
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}
type Slots struct {
	//DoctorID uint      `json:"doctorID"`
	SlotDay time.Time `json:"specificDay"`
}

//Reference: https://stackoverflow.com/questions/38501646/initialize-nested-struct-definition-in-golang-if-it-have-same-objects

type AutoGenerated struct {
	DoctorID     uint          `json:"doctorID"`
	Doctor       models.Doctor `json:"doctor"`
	Appointments uint          `json:"frequency,omitempty"`
}

type AutoHours struct {
	DoctorID uint          `json:"doctorID"`
	Doctor   models.Doctor `json:"doctor"`
	Hours    float64       `json:"hours"`
}

type DoctorAvailable struct {
	Doctor       uint     `json:"doctorID"`
	Availability []string `gorm:"type:text" json:"availability"`
}

// GET function to view lists of doctors in the system.
// /doctors
func ViewAllDoctors(v *gin.Context) {
	var docs []models.Doctor
	role := v.GetString("role")
	fmt.Println(role)

	//If the user is a patient or a doctor, they can view the list of doctors
	if role == "patient" || role == "doctor" {

		models.DB.Raw("SELECT first_name, last_name, email FROM hospital.doctors").Find(&docs)
	} else {
		models.DB.Raw("SELECT * FROM hospital.doctors").Find(&docs)
	}
	v.JSON(http.StatusOK, gin.H{"doctors": docs})
}

//SQL command references
//https://www.w3resource.com/sql/where-clause.php
//https://www.linkedin.com/pulse/hospital-management-system-project-oracle-sql-md-almuntsir

//GET function to view all the requested doctors with their id in the system
// /doctors/:id
func ViewRequestedDoctor(r *gin.Context) {
	var doc models.Doctor
	role := r.GetString("role")

	if role == "patient" || role == "doctor" {
		if err := models.DB.Raw("SELECT first_name, last_name, email FROM hospital.doctors WHERE id=?", r.Param("id")).First(&doc).Error; err != nil {
			r.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!!"})
			return
		}
	} else {
		if err := models.DB.Raw("SELECT * FROM hospital.doctors WHERE id=?", r.Param("id")).First(&doc).Error; err != nil {
			r.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!!!"})
			return
		}
	}

	r.JSON(http.StatusOK, gin.H{"doctor": doc})
}

//POST function to view doctor available slots
// /doctors/:id/slots
func ViewDoctorSlots(c *gin.Context) {
	var doc models.Doctor
	var input Slots
	var givenSched []models.Appointment

	if err := models.DB.Where("id = ?", c.Param("id")).First(&doc).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!"})
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	year, month, day := input.SlotDay.Date()

	//9AM
	DayStartTime := time.Date(year, month, day, 9, 0, 0, 0, time.UTC)
	//5PM
	DayEndTime := time.Date(year, month, day, 17, 0, 0, 0, time.UTC)

	models.DB.Raw("SELECT * FROM hospital.appointments WHERE hospital.appointments.doctor_id=? AND DATE(hospital.appointments.start_time)=? order by start_time asc", c.Param("id"), string(input.SlotDay.Format("2006-01-02"))).Find(&givenSched)

	//Condition if there are no appointments, then doctor is available from 9AM TO 5PM
	if len(givenSched) == 0 {
		doc.Availability = nil
		doc.Availability = append(doc.Availability, string(DayStartTime.String()+"---"+DayEndTime.String()))
	} else if len(givenSched) == 1 {
		if !DayStartTime.After(givenSched[0].StartTime) {
			doc.Availability = append(doc.Availability, string(DayStartTime.String()+"---"+(givenSched[0]).StartTime.String()))
			doc.Availability = append(doc.Availability, (givenSched[0]).EndTime.String()+"---"+string(DayEndTime.String()))
		}
	} else if len(givenSched) > 1 {
		for i := 0; i < len(givenSched); i++ {
			t := (givenSched[i]).StartTime
			//End time of one appointment
			t1 := (givenSched[i]).EndTime

			//Handling final element
			if i == len(givenSched)-1 {
				doc.Availability = append(doc.Availability, string(t1.String()+"---"+string(DayEndTime.String())))
				break
			}

			//Start time of next appointment
			t2 := (givenSched[i+1]).StartTime
			if i == 0 {
				doc.Availability = append(doc.Availability, string(DayStartTime.String()+"---"+t.String()))
				doc.Availability = append(doc.Availability, string(t1.String()+"---"+t2.String()))
			} else {
				doc.Availability = append(doc.Availability, string(t1.String()+"---"+t2.String()))
			}

		}

	}
	models.DB.Model(&doc).Select("availability").Updates(doc.Availability)
	c.JSON(http.StatusOK, gin.H{"availability": doc.Availability})
}

//POST function to view availability of all Doctors
// /doctors//availability/all
func ViewDoctorAvailability(c *gin.Context) {
	var docs []models.Doctor
	var result []DoctorAvailable
	var input Slots
	var givenSched []models.Appointment

	role := c.GetString("role")

	if (role != "admin") && (role != "patient") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only admins and patients are authorized to see ALL doctors' availabilities!"})
		return
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	year, month, day := input.SlotDay.Date()

	//9AM
	DayyStartTime := time.Date(year, month, day, 9, 0, 0, 0, time.UTC)
	//5PM
	DayyEndTime := time.Date(year, month, day, 17, 0, 0, 0, time.UTC)

	//A for loop for iterating for all the doctors
	for a := 0; a < len(docs); a++ {

		models.DB.Raw("SELECT * FROM hospital.appointments WHERE hospital.appointments.doctor_id=? AND DATE(hospital.appointments.start_time)=? AND appointments.deleted_at IS NULL order by start_time asc", docs[a].ID, string(input.SlotDay.Format("2006-01-02"))).Find(&givenSched)

		//if NO APPOINTMENTS, the doctor is available from 9AM to 5PM
		if len(givenSched) == 0 {
			docs[a].Availability = nil
			docs[a].Availability = append(docs[a].Availability, string(DayyStartTime.String()+"---"+DayyEndTime.String()))

		} else if len(givenSched) == 1 {

			if !DayyStartTime.After(givenSched[0].StartTime) {
				docs[a].Availability = append(docs[a].Availability, string(DayyStartTime.String()+"---"+(givenSched[0]).StartTime.String()))
				docs[a].Availability = append(docs[a].Availability, (givenSched[0]).EndTime.String()+"---"+string(DayyEndTime.String()))
			}
		} else if len(givenSched) > 1 {

			for i := 0; i < len(givenSched); i++ {
				t := (givenSched[i]).StartTime //start time of one appointment
				t1 := (givenSched[i]).EndTime  //end time of one appointment

				if i == len(givenSched)-1 {

					//handle final element in this code
					docs[a].Availability = append(docs[a].Availability, string(t1.String()+"---"+string(DayyEndTime.String())))
					break
				}

				t2 := (givenSched[i+1]).StartTime //start time of next appointment
				if i == 0 {
					docs[a].Availability = append(docs[a].Availability, string(DayyStartTime.String()+"---"+t.String()))
					docs[a].Availability = append(docs[a].Availability, string(t1.String()+"---"+t2.String()))
				} else {
					docs[a].Availability = append(docs[a].Availability, string(t1.String()+"---"+t2.String()))
				}

			}

		}

		models.DB.Model(&docs[a]).Select("availability").Updates(docs[a].Availability)

		result = append(result, DoctorAvailable{uint(docs[a].ID), docs[a].Availability})
	}

	c.JSON(http.StatusOK, gin.H{"availability": result})
}

//POST function to view doctors with most appointments in a given day
// /doctors/most/appointments
func ViewDoctorsMostAppointments(c *gin.Context) {
	var input Slots
	var doc models.Doctor
	var doctorID uint
	var freq uint
	var result []AutoGenerated

	role := c.GetString("role")

	if role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wrong user! Only admins are able to check doctors with most appointments!"})
		return
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dup1 := input.SlotDay.Format("2006-01-02")
	rows, err := models.DB.Raw("Select doctor_id, count(*) From hospital.appointments WHERE DATE(start_time) = ? AND deleted_at IS NULL Group By doctor_id order by count(*) desc", string(dup1)).Rows()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Opps, no appointments in the record! Try again!"})
		return
	}

	for rows.Next() {
		err := rows.Scan(&doctorID, &freq)
		if err != nil {
			log.Fatal(err)
		}
		if err := models.DB.Where("id = ?", doctorID).Find(&doc).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"Error": "Doctor not found!"})
			return
		}
		result = append(result, AutoGenerated{doctorID, doc, freq})
	}

	c.JSON(http.StatusOK, gin.H{"Doctors with most appointments are on this day!": result})
}

//POST function to view doctors who have more than 6 hours total appointments in a day
// /doctors/most/hours
func ViewDoctorsMostHours(c *gin.Context) {
	var input Slots
	var doc models.Doctor
	var doctorID uint
	var freq float64
	var result []AutoHours

	role := c.GetString("role")

	if role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Wrong user! Only admins are able to check doctors with most appointments!"})
		return
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
		return
	}

	dup1 := input.SlotDay.Format("2006-01-02")

	//Creating the logic in the database
	rows, err := models.DB.Raw("Select doctor_id, SUM((TIMEDIFF(end_time, start_time)/10000)) From hospital.appointments WHERE DATE(start_time) = ? Group By doctor_id HAVING SUM((TIMEDIFF(end_time, start_time)/10000))>6 order by SUM((TIMEDIFF(end_time, start_time)/10000)) desc", string(dup1)).Rows()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"Error": "Opps, No appointments in the record! Try again!"})
		return
	}

	for rows.Next() {
		err := rows.Scan(&doctorID, &freq)
		if err != nil {
			log.Fatal(err)
		}

		if err := models.DB.Where("id = ?", doctorID).Find(&doc).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Doctor not found!"})
			return
		}
		result = append(result, AutoHours{doctorID, doc, freq})
	}
	if result == nil {
		c.JSON(http.StatusOK, gin.H{"Error": "No doctors have more than 6 hours for this day!!"})
	} else {

		c.JSON(http.StatusOK, gin.H{"Doctors with more than 6 hours on this day!!": result})
	}
}
